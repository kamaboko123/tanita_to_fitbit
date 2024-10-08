# Tanita to Fitbit Data Transfer
This is a simple script to transfer data from a Tanita scale to Fitbit.  
It uses the Tanita API to get the data and the Fitbit API to upload it.

## Setup

### Build
```bash
make
```

### Configuration
#### Register application for Fitbit API

1. Go to the [Fitbit Developer Console](https://dev.fitbit.com/apps/new)
2. Register a new application
3. Fill in the required fields


#### Register application for HealthPlanet API

1. Go to the [HealthPlanet](https://www.healthplanet.jp/apis_account.do)
2. Register a new application
3. Fill in the required fields


#### Configuration file
Copy the `config.example.json` to `config.json` and fill in the required fields.

```bash
cp config.example.json config.json
vim config.json
```

#### Setup first token of Fitbit API
Create token file(fb_token.json) for Fitbit API
```bash
./tanita-to-fitbit -m init_fitbit
```

Get access token from [here]("https://dev.fitbit.com/build/reference/web-api/troubleshooting-guide/oauth2-tutorial/").
and write access_token, refresh_token, user_id and etc to `fitbit_token.json`.

```bash
vim fb_token.json
```
```json
{
  "access_token": "***********************",
  "refresh_token": "***********************",
  "expires_in": 28800,
  "scope": "heartrate nutrition electrocardiogram respiratory_rate irregular_rhythm_notifications oxygen_saturation activity weight social settings temperature location sleep cardio_fitness profile",
  "token_type": "Bearer",
  "user_id": "*****"
}
```


#### Setup first token of Tanita API
Auth to HealthPlanet API
```bash
./tanita-to-fitbit -m init_healthplanet
```

The script show auth URL and Please access it and get code.
Next, put the code to the terminal.

token file(hp_token.json) will be created and finish the setup.



## Usage
Get BodyWeight and BodyFat data from Tanita(HealthPlanet) and upload to Fitbit.

```bash
./tanita-to-fitbit -m sync
```

