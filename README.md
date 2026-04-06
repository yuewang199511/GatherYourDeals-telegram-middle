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

[docs/outbound_settings.md](docs/outbound_settings.md)docs/outbound_settings.md