package health_planet

import (
    "bufio"
    "fmt"
    "os"
    "io/ioutil"
    "net/http"
    "net/url"
    "encoding/json"
    "errors"
    "time"
    "strconv"
    "log/slog"
)

type Auth struct {
    url string
    client_id string
    client_secret string
    token *Token
    dump_filepath string
    Logger *slog.Logger
}

type Token struct {
    AccessToken string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn int64 `json:"expires_in"`
    
    // metadata
    Create_date int64 `json:"create_date", omitempty`
}


type Client struct {
    url string
    auth *Auth
    Logger *slog.Logger
    Timezone *time.Location
}


type InnerscanResponse struct {
    BirthDate string `json:"birth_date"`
    Height string `json:"height"`
    Sex string `json:"sex"`
    Data []struct{
        Date string `json:"date"`
        KeyData string `json:"keydata"`
        Tag string `json:"tag"`
        Model string `json:"model"`
    } `json:"data"`
}

type InnerscanData struct {
    Date time.Time
    Weight float64
    BodyFat float64
}

func (d *InnerscanData) String() string {
    return fmt.Sprintf("(%s)Weight: %f, BodyFat: %f", d.Date, d.Weight, d.BodyFat)
}

type InnerscanDataMap map[string]*InnerscanData

const TokenRefreshThreshold = 60 * 60 * 24 * 7 // 1 week

func NewAuth(url string, client_id string, client_secret string, dump_filepath string, logger *slog.Logger) *Auth {
    auth := Auth{
        url: url,
        client_id: client_id,
        client_secret: client_secret,
        dump_filepath: dump_filepath,
        token: nil,
        Logger: logger,
    }
    auth.token = &Token{Create_date: 0}

    return &auth
}

func (a *Auth) DumpToken() error {
    data, err := json.MarshalIndent(a.token, "", "  ")
    if err != nil {
        return err
    }

    f, err := os.OpenFile(a.dump_filepath, os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = f.Write(data)
    if err != nil {
        return err
    }

    return nil
}

func (t *Token) IsTokenExpired() bool {
    return t.Create_date + t.ExpiresIn < time.Now().Unix()
}

func (t *Token) IsTokenNeedRefresh() bool {
    return t.Create_date + t.ExpiresIn - TokenRefreshThreshold < time.Now().Unix()
}



func (a *Auth) LoadToken() error {
    data, err := ioutil.ReadFile(a.dump_filepath)
    if err != nil {
        return err
    }

    a.token = &Token{}
    err = json.Unmarshal(data, a.token)
    if err != nil {
        return err
    }

    return nil
}

func (a *Auth) IsTokenValid() bool {
    if a.token == nil {
        return false
    }
    if a.token.Create_date == 0 {
        return false
    }

    return !a.token.IsTokenExpired()
}

func (a *Auth) GetAuthURL() (string, error) {
    u, err := url.Parse(a.url)
    if err != nil {
        return "", err
    }

    u.Path = "/oauth/auth"

    q := u.Query()
    q.Set("client_id", a.client_id)
    q.Set("client_secret", a.client_secret)
    q.Set("redirect_uri", "http://localhost")
    q.Set("response_type", "code")
    q.Set("scope", "innerscan,sphygmomanometer,pedometer,smug")
    u.RawQuery = q.Encode()

    return u.String(), nil
}

func (a *Auth) GetToken(code string) (*Token, error) {
    u, err := url.Parse(a.url)
    if err != nil {
        return nil, err
    }

    u.Path = "/oauth/token"
    q := u.Query()
    q.Set("client_id", a.client_id)
    q.Set("client_secret", a.client_secret)
    q.Set("redirect_uri", "http://localhost")
    q.Set("grant_type", "authorization_code")
    q.Set("code", code)
    u.RawQuery = q.Encode()

    req, err := http.NewRequest("POST", u.String(), nil)
    if err != nil {
        return nil, err
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode != 200 {
        return nil, errors.New("[HealthPlanet]Failed to get token")
    }

    body, _ := ioutil.ReadAll(resp.Body)
    token := Token{}
    err = json.Unmarshal(body, &token)
    if err != nil {
        return nil, err
    }
    a.token = &token
    a.token.Create_date = time.Now().Unix()
    

    return &token, nil
}

func (a *Auth) RefreshToken() error{
    if !a.token.IsTokenNeedRefresh(){
        return nil
    }
    fmt.Println("Token is need to refresh")

    if a.token == nil {
        return errors.New("[HealthPlanet]Token is not initialized")
    }

    u, err := url.Parse(a.url)
    if err != nil {
        return err
    }

    u.Path = "/oauth/token"
    q := u.Query()
    q.Set("client_id", a.client_id)
    q.Set("client_secret", a.client_secret)
    q.Set("redirect_uri", "http://localhost")
    q.Set("grant_type", "refresh_token")
    q.Set("refresh_token", a.token.RefreshToken)
    u.RawQuery = q.Encode()

    req, err := http.NewRequest("POST", u.String(), nil)
    if err != nil {
        return err
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        return errors.New("[HealthPlanet]Failed to refresh token")
    }

    body, _ := ioutil.ReadAll(resp.Body)
    err = json.Unmarshal(body, a.token)
    if err != nil {
        return err
    }
    fmt.Println("Success to refresh token")

    err = a.DumpToken()
    if err != nil {
        return err
    }

    return nil
}


func (a *Auth) InitToken() error{
    // check dump file exists
    _, err := os.Stat(a.dump_filepath)
    if err == nil {
        return errors.New("[HealthPlanet]Token file already exists. If you want to reinitilize, please remove the file")
    }

    url, err := a.GetAuthURL()
    if err != nil {
        return err
    }
    fmt.Printf("Access to: %s\n", url)

    fmt.Printf("and enter the code:")
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    code := scanner.Text()

    _, err = a.GetToken(code)
    if err != nil {
        return err
    }
    
    err = a.DumpToken()
    if err != nil {
        return err
    }

    fmt.Println("Success to init token")
    return nil
}


func NewClient(url string, auth *Auth, logger *slog.Logger, timezone *time.Location) *Client{
    return &Client{url: url, auth: auth, Logger: logger, Timezone: timezone}
}

func (c *Client) GetInnerscanData() (InnerscanDataMap, error){
    u, err := url.Parse(c.url)
    if err != nil {
        return nil, err
    }
    
    current_date := time.Now().Add(-24 * 7 * time.Hour)
    current_date_str := current_date.Format("20060102150405")

    u.Path = "/status/innerscan.json"
    q := u.Query()
    q.Set("access_token", c.auth.token.AccessToken)
    q.Set("from", current_date_str)
    q.Set("tag", "6021,6022") // 6021: Weight, 6022: Body Fat

    u.RawQuery = q.Encode()

    c.Logger.Debug(fmt.Sprintf("Access to: %s", u.String()))

    req, err := http.NewRequest("GET", u.String(), nil)
    if err != nil {
        return nil, err
    }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode != 200 {
        return nil, errors.New("[HealthPlanet]Failed to get innerscan data")
    }

    body, _ := ioutil.ReadAll(resp.Body)
    c.Logger.Debug(fmt.Sprintf("Response: %s", body))
    resp_data := InnerscanResponse{}
    err = json.Unmarshal(body, &resp_data)
    if err != nil {
        return nil, err
    }

    return(resp_data.GetInnerscanDataMap(c.Timezone))
}

func (resp *InnerscanResponse) GetInnerscanDataMap(timezone *time.Location) (InnerscanDataMap, error) {
    ret := make(InnerscanDataMap)
    
    for _, d := range resp.Data {
        if _, ok := ret[d.Date]; !ok {
            ret[d.Date] = &InnerscanData{}
        }
        date, err := time.ParseInLocation("200601021504", d.Date, timezone)
        if err != nil {
            return nil, err
        }
        ret[d.Date].Date = date

        if d.Tag == "6021" {
            value, err := strconv.ParseFloat(d.KeyData, 64)
            if err != nil {
                return nil, err
            }
            ret[d.Date].Weight = value
        }
        if d.Tag == "6022" {
            value, err := strconv.ParseFloat(d.KeyData, 64)
            if err != nil {
                return nil, err
            }
            ret[d.Date].BodyFat = value
        }
    }

    return ret, nil
}

