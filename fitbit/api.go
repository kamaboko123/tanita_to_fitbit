package fitbit

import (
    "strings"
    "fmt"
    "net/url"
    "net/http"
    "io/ioutil"
    "os"
    "encoding/json"
    "errors"
    "time"
    "log/slog"
)

type Auth struct {
    url string
    client_id string
    client_secret string

    token *Token
    dump_filepath string
}


type Token struct {
    Access_token string `json:"access_token"`
    Refresh_token string `json:"refresh_token"`
    Expires_in int64 `json:"expires_in"`
    Scope string `json:"scope"`
    Token_type string `json:"token_type"`
    User_id string `json:"user_id"`

    Create_date int64 `json:"create_date", omitempty`
}


type Client struct {
    url string
    auth *Auth
    logger *slog.Logger
    Timezone *time.Location
}


type WeightLogResponse struct {
    Weight []struct {
        Bmi float64 `json:"bmi"`
        Date string `json:"date"`
        Fat float64 `json:"fat"`
        LogId int64 `json:"logId"`
        Source string `json:"source"`
        Time string `json:"time"`
        Weight float64 `json:"weight"`
    } `json:"weight"`
}

type WeightLog struct {
    Date time.Time
    Weight float64
    Fat float64
}

func (w *WeightLogResponse) ToWeightLog(timezone *time.Location) ([]WeightLog, error) {
    var weight_logs []WeightLog
    for _, wl := range w.Weight {
        date, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", wl.Date, wl.Time), timezone)
        if err != nil {
            return nil, err
        }

        weight_logs = append(weight_logs, WeightLog{Date: date, Weight: wl.Weight, Fat: wl.Fat})
    }

    return weight_logs, nil
}

func (w *WeightLog) String() string {
    return fmt.Sprintf("(%s)Weight: %f, Fat: %f", w.Date, w.Weight, w.Fat)
}


func NewAuth(url string, client_id string, client_secret string, dump_filepath string) *Auth {
    auth := Auth{
        url: url,
        client_id: client_id,
        client_secret: client_secret,
        dump_filepath: dump_filepath,
        token: nil,
    }
    auth.token = &Token{Create_date: 0}

    return &auth
}

func (a *Auth) InitToken() error {
    if _, err := os.Stat(a.dump_filepath); err == nil {
        return errors.New("Token file already exists. If you want to reinitialize, please remove token file")
    }

    // create token template file
    a.token.Access_token = "PUT_YOUR_ACCESS_TOKEN"
    a.token.Refresh_token = "PUT_YOUR_REFRESH_TOKEN"
    a.token.Expires_in = 0
    a.token.Scope = "PUT_YOUR_SCOPE"
    a.token.Token_type = "Bearer"
    a.token.User_id = ""
    a.token.Create_date = 0

    err := a.DumpToken()
    if err != nil {
        return err
    }

    u, err := url.Parse("https://dev.fitbit.com/build/reference/web-api/troubleshooting-guide/oauth2-tutorial/")
    if err != nil {
        return err
    }

    q := u.Query()
    q.Set("clientEncodedId", a.client_id)
    u.RawQuery = q.Encode()

    fmt.Println("Get Access token from Fitbit tutorial page:" + u.String())
    fmt.Println("Application Type: Client")
    fmt.Println("Callback URL: http://localhost")
    fmt.Println("Next, edit token file(place: access_token, refresh_token, expires_in, scope, token_type, user_id): " + a.dump_filepath)

    return nil
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

func (a *Auth) RefreshToken() error {
    u, err := url.Parse(a.url)
    if err != nil {
        return err
    }

    u.Path = "/oauth2/token"
    q := u.Query()
    q.Set("client_id", a.client_id)
    q.Set("grant_type", "refresh_token")
    q.Set("refresh_token", a.token.Refresh_token)
    u.RawQuery = q.Encode()

    req, err := http.NewRequest("POST", u.String(), nil)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return errors.New("[fitbit]Failed to get token")
    }

    body, _ := ioutil.ReadAll(resp.Body)
    err = json.Unmarshal(body, a.token)
    if err != nil {
        return err
    }
    a.token.Create_date = time.Now().Unix()

    err = a.DumpToken()
    if err != nil {
        return err
    }

    return nil
}


func NewClient(url string, auth *Auth, logger *slog.Logger, timezone *time.Location) *Client {
    return &Client{url: url, auth: auth, logger:logger, Timezone: timezone}
}

func (c *Client) GetWeightLog(date time.Time) (*WeightLogResponse, error) {
    u, err := url.Parse(c.url)
    if err != nil {
        return nil, err
    }
    
    _path := "/1/user/[user-id]/body/log/weight/date/[date].json"
    _path = strings.Replace(_path, "[user-id]", c.auth.token.User_id, -1)
    _path = strings.Replace(_path, "[date]", date.Format("2006-01-02"), -1)

    u.Path = _path

    c.logger.Debug(fmt.Sprintf("[fitbit]Get weight log: %s", u.String()))
    
    client := &http.Client{}
    req, err := http.NewRequest("GET", u.String(), nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer " + c.auth.token.Access_token)
    req.Header.Set("accept", "application/json")
    req.Header.Set("accept-language", "ja_JP")
    req.Header.Set("accept-locale", "ja_JP")

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    if resp.StatusCode != 200 {
        return nil, errors.New(fmt.Sprintf("[fitbit]Failed to get weight log: (%d) %s", resp.StatusCode, body))
    }

    weight_log := WeightLogResponse{}
    c.logger.Debug(fmt.Sprintf("[fitbit]Response: %s", body))
    err = json.Unmarshal(body, &weight_log)

    return &weight_log, nil
}

func (c *Client) CreateWeightAndFatLog(date time.Time, weight float64, fat float64) error {
    err := c.CreateWeightLog(date, weight)
    if err != nil {
        return err
    }

    // TODO: rollback if failed to create fat log

    err = c.CreateFatLog(date, fat)
    if err != nil {
        return err
    }

    return nil
}


func (c *Client) CreateWeightLog(date time.Time, weight float64) error {
    u, err := url.Parse(c.url)
    if err != nil {
        return err
    }

    u.Path = "/1/user/[user-id]/body/log/weight.json"
    u.Path = strings.Replace(u.Path, "[user-id]", c.auth.token.User_id, -1)

    // timezoneどうなってんのかわからん
    q := u.Query()
    q.Set("date", date.Format("2006-01-02"))
    q.Set("weight", fmt.Sprintf("%f", weight))
    q.Set("time", date.Format("15:04:05"))
    u.RawQuery = q.Encode()

    client := &http.Client{}
    req, err := http.NewRequest("POST", u.String(), nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer " + c.auth.token.Access_token)
    req.Header.Set("accept", "application/json")
    req.Header.Set("accept-language", "ja_JP")
    req.Header.Set("accept-locale", "ja_JP")
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    //body, _ := ioutil.ReadAll(resp.Body)
    //fmt.Println(string(body))
    
    if resp.StatusCode != 201 {
        return errors.New("[fitbit]Failed to create weight log")
    }

    return nil
}

func (c *Client) CreateFatLog(date time.Time, fat float64) error {
    u, err := url.Parse(c.url)
    if err != nil {
        return err
    }

    u.Path = "/1/user/[user-id]/body/log/fat.json"
    u.Path = strings.Replace(u.Path, "[user-id]", c.auth.token.User_id, -1)

    // timezoneどうなってんのかわからん
    q := u.Query()
    q.Set("date", date.Format("2006-01-02"))
    q.Set("fat", fmt.Sprintf("%f", fat))
    q.Set("time", date.Format("15:04:05"))
    u.RawQuery = q.Encode()

    client := &http.Client{}
    req, err := http.NewRequest("POST", u.String(), nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer " + c.auth.token.Access_token)
    req.Header.Set("accept", "application/json")
    req.Header.Set("accept-language", "ja_JP")
    req.Header.Set("accept-locale", "ja_JP")
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    //body, _ := ioutil.ReadAll(resp.Body)
    //fmt.Println(string(body))

    if resp.StatusCode != 201 {
        return errors.New("[fitbit]Failed to create weight log")
    }

    return nil
}
