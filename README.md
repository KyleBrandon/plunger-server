# Plunge Server

##### Control your plunge tub with a Raspberry Pi

## Configuration

### Environment Variables

The following environment variables are used when configuring the Plunger Server.

| Property             | Type           | Description                                               |
| -------------------- | -------------- | --------------------------------------------------------- |
| DATABASE_URL         | string         | The connection string used to connect to the database.    |
| PORT                 | number         | The port used by the Plunger API server.                  |
| LOG_FILE_LOCATION    | string         | The path to the location to store the log file.           |
| CONFIG_FILE_LOCATION | string         | The path to the location to store the configuration file. |
| TWILIO_ACCOUNT_SID   | string \| null | The account identifer from the Twilio admin console.      |
| TWILIO_AUTH_TOKEN    | string \| null | The authentication token from the Twilio admin console.   |
| TWILIO_FROM_PHONE_NO | string \| null | The phone number in Twilio to send SMS messages from.     |
| TWILIO_TO_PHONE_NO   | string \| null | The phone number to send SMS messages to.                 |

#### Examples

```sh
 DATABASE_URL="postgres://<username>:<password>@<IP address>:<port>/<database name>?sslmode=disable"
 PORT=8080
 LOG_FILE_LOCATION="./plunger-server.log"
 CONFIG_FILE_LOCATION="./config/config.json"
 TWILIO_ACCOUNT_SID="<account identifier>"
 TWILIO_AUTH_TOKEN="<auth token for the account>"
 TWILIO_FROM_PHONE_NO="11235551212"
 TWILIO_TO_PHONE_NO="12345551212"
```
