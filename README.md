# Reviewpad API

## Run project

| Pre-requirements: [go](https://go.dev/) must be installed. |
| ---------------------------------------------------------- |

1. Checkout code 
```bash
git clone git@github.com:reviewpad/jubilant-api.git
```

2. Install project
```bash
go get
```

3. Build project
```bash
go build
```

4. Run project
```bash
./api
```

You now have the project running on `localhost:8080`.

## Curl Request Example

```bash
curl --location --request POST 'http://localhost:8080/dry-run' \
--header 'Content-Type: application/json' \
--data-raw '{
    "gitHubToken": "GITHUB_TOKEN",
    "pullRequestUrl": "https://github.com/google/guava/pull/6059",
    "reviewpadConfiguration": "api-version: reviewpad.com/v1.x\n\nmode: verbose\n\nrules:\n  tautology:\n    kind: patch\n    description: Always true\n    spec: 1 == 1\n\nworkflows:\n  - name: say-hello\n    description: Say Hello World\n    if:\n      - rule: tautology\n        extra-actions:\n          - $comment(\"Hello World\")\n"
}'
```