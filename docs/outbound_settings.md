# outbound settings

For all related downstream services, this document will list the environment variable required to setup the connection and the API documentation to refer to.

# gatherYourDeals-data service

environment variable: GYD_DATA_URL
api document: [text](https://github.com/yuewang199511/GatherYourDeals-data/blob/main/docs/api/api.yaml)

# gatherYourDeals-ETL service

environment variable: GYD_ETL_URL
api document: 

```yaml
openapi: 3.0.3
info:
  title: ETL Service API
  description: >
    Internal ETL service that accepts a remote address and processes it
    synchronously. No authentication required.
  version: 1.0.0

servers:
  - url: https://etl-service.yourdomain.com/v1

paths:
  /etl:
    post:
      summary: Run an ETL process from a remote address
      description: >
        Triggers a synchronous ETL process from the given remote address.
        Blocks until the process completes and returns success or failure.
      operationId: runEtl
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/EtlRequest'
            example:
              source: "https://datasource.example.com/data.csv"
      responses:
        '200':
          description: ETL completed successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EtlResponse'
              example:
                success: true
                message: "ETL completed successfully"
        '400':
          description: Bad request — missing or invalid source address
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EtlResponse'
              example:
                success: false
                message: "source address must not be empty"
        '422':
          description: ETL failed — source was reachable but processing failed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EtlResponse'
              example:
                success: false
                message: "Failed to parse data from source"
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EtlResponse'
              example:
                success: false
                message: "Unexpected error during ETL process"

components:
  schemas:
    EtlRequest:
      type: object
      required:
        - source
      properties:
        source:
          type: string
          description: >
            Remote address to pull data from.
            Can be any format — URL, S3 path, file path, etc.
          example: "https://datasource.example.com/data.csv"

    EtlResponse:
      type: object
      required:
        - success
        - message
      properties:
        success:
          type: boolean
          description: Whether the ETL process completed successfully
        message:
          type: string
          description: Human-readable result or error description
          example: "ETL completed successfully"
```

# gatherYourDeals-llm-chatbot service

environment variable: GYD-LLM-CHATBOT-URL
api document: 

```yaml
openapi: 3.0.3
info:
  title: LLM Service API
  description: >
    Stateless LLM service that accepts conversation history and a JWT for
    downstream database authentication. All session and history management
    is handled by the caller (Bot Gateway or Frontend).
  version: 1.0.0

servers:
  - url: https://llm-service.yourdomain.com/v1

security:
  - BearerAuth: []

paths:
  /chat:
    post:
      summary: Send a conversation to the LLM
      description: >
        Accepts the full conversation history and returns the assistant reply.
        The JWT is forwarded as-is to the downstream database service.
        Caller is responsible for appending the reply to history after this call.
      operationId: chat
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ChatRequest'
            example:
              messages:
                - role: user
                  content: "What is Redis?"
                - role: assistant
                  content: "Redis is an in-memory key-value store."
                - role: user
                  content: "How is it different from Postgres?"
      responses:
        '200':
          description: Successful LLM response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ChatResponse'
              example:
                message:
                  role: assistant
                  content: "Redis is in-memory and schema-less, while Postgres is disk-based and relational."
                stop_reason: end_turn
                usage:
                  input_tokens: 210
                  output_tokens: 85
        '400':
          description: Bad request — invalid message format or missing fields
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                code: invalid_request
                message: "messages array must not be empty"
        '401':
          description: Unauthorized — missing or invalid JWT
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                code: unauthorized
                message: "Missing or invalid Authorization header"
        '419':
          description: Token expired — JWT was valid but has expired
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                code: token_expired
                message: "Access token has expired"
        '422':
          description: Unprocessable — e.g. messages exceed token limit
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                code: context_too_long
                message: "Conversation history exceeds maximum token limit"
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                code: internal_error
                message: "Unexpected error occurred"

  /health:
    get:
      summary: Health check
      description: Returns service health status. Does not require authentication.
      operationId: healthCheck
      security: []
      responses:
        '200':
          description: Service is healthy
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/HealthResponse'
              example:
                status: ok

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: >
        JWT issued by your auth service. This token is forwarded to the
        downstream database service on every request. The LLM service
        does not validate claims beyond basic signature verification.

  schemas:
    Message:
      type: object
      required:
        - role
        - content
      properties:
        role:
          type: string
          enum:
            - user
            - assistant
          description: Who sent this message
        content:
          type: string
          description: Text content of the message

    ChatRequest:
      type: object
      required:
        - messages
      properties:
        messages:
          type: array
          description: >
            Full conversation history in chronological order.
            Must alternate between user and assistant roles.
            Last entry must be role: user.
          minItems: 1
          items:
            $ref: '#/components/schemas/Message'

    ChatResponse:
      type: object
      required:
        - message
        - stop_reason
        - usage
      properties:
        message:
          $ref: '#/components/schemas/Message'
          description: The assistant reply to append to history
        stop_reason:
          type: string
          enum:
            - end_turn
            - max_tokens
            - error
          description: >
            Why the LLM stopped generating.
            Caller should warn user if stop_reason is max_tokens as reply may be cut off.
        usage:
          $ref: '#/components/schemas/Usage'

    Usage:
      type: object
      required:
        - input_tokens
        - output_tokens
      properties:
        input_tokens:
          type: integer
          description: Tokens consumed by the input messages
          example: 210
        output_tokens:
          type: integer
          description: Tokens consumed by the assistant reply
          example: 85

    HealthResponse:
      type: object
      required:
        - status
      properties:
        status:
          type: string
          enum:
            - ok
            - degraded
          example: ok

    ErrorResponse:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: string
          description: Machine-readable error code
          enum:
            - invalid_request
            - unauthorized
            - token_expired
            - context_too_long
            - internal_error
          example: unauthorized
        message:
          type: string
          description: Human-readable error description
          example: "Missing or invalid Authorization header"
```

# redis

environment variable: REDIS_URL


