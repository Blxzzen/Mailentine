# 💌 &nbsp; Mailentine &nbsp; 💌
>*A Valentines Day email service your partner never asked for !*


\
Mailentine is a web service that allows you to send emails to a friend or partner automatically every day. It was developed as a Valentines Day gift, for a person who likes receiving heart felt messages. While Mailentine is developed to be as *"plug and use"* as possible, you will still need your own `.env` file and `message.json` file. This `README` will provide a walkthrough on how to set this service up completely for free.<br><br><br>

## Quick Start Guide:
<br>

1. Install Go (<a href="https://go.dev/doc/install" target="_blank">official installation guide</a>)

2. Clone the Mailentine Repository:
    ```bash
    git clone https://github.com/Blxzzen/Mailentine.git
    ```

3. Set up your `.env` like so:
    ```js
    SENDER_EMAIL=youremail@gmail.com
    SENDER_PASS=yourgoogleapppassword
    RECEIVER_EMAIL=receiveremail@gmail.com
    PORT=port
    USER=httpusername
    PASS=httppassword
    ```

4. Edit `example-messages.json` by renaming file to `messages.json` and deleting comment line, then add custom messages.

5. Install Dependencies:
    ```go
    go mod tidy 
    ```

6. Run the App: Start the service with:
    ```go 
    go run main.go
    ``` 
    or build with go build and run the binary.

7. Use Endpoints: Access `/` for a health check and `/send-email` (HTTP Auth-protected), to trigger email sending.