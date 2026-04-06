# GatherYourDeals-telegram-middle

This is a go 1.25.5 based gin service that connect telegram chatbot.

Use github.com/go-telegram-bot-api/telegram-bot-api to talk with chatbot

# supported commands from chatbot

- /start: show "This is the gatherYourDeals bot! Right now it is only a alpha version! try /help to see what you can do!"
- /help: show available commands
- /login: login for the GatherYourDealsService. If not logged in or log in expired, all other operations except /help will show warning to login
- /logout: call the service to logout
- /etl: follow with a shared link of google drive picture of folder of pictures. Transform into item records. right now only accept receipts. 
- other messages: Up to 5 rounds of memory, those messages will be sent to the LLM chatbot to answer questions related to your purchase records.

# Environment setup

See [docs/outbound_settings.md](docs/outbound_settings.md) for downstream service API details.

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | — | Telegram bot token from [@BotFather](https://t.me/botfather) |
| `GYD_DATA_URL` | yes | — | Base URL of the GatherYourDeals data service |
| `GYD_ETL_URL` | yes | — | Base URL of the ETL service |
| `GYD_LLM_CHATBOT_URL` | yes | — | Base URL of the LLM chatbot service |
| `REDIS_URL` | yes | — | Redis connection URL, e.g. `redis://localhost:6379` |
| `PORT` | no | `8080` | Port the HTTP server listens on |
| `CB_FAILURE_THRESHOLD` | no | `5` | Consecutive failures before opening the circuit breaker |
| `CB_SUCCESS_THRESHOLD` | no | `2` | Consecutive successes in half-open state before closing |
| `CB_OPEN_TIMEOUT` | no | `30s` | How long the circuit stays open before probing (e.g. `30s`, `1m`) |