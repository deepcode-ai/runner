# runner
DeepCode self-hosted runners.  The power of DeepCode Cloud with the safety of your infrastructure.
[![DeepCode](https://app.deepcode.com/gh/deepcode-ai/runner.svg/?label=active+issues&show_trend=true&token=Cs-Qwzy6mON1zpyhLBHdzRC_)](https://app.deepcode.com/gh/deepcode-ai/runner/?ref=repository-badge)

## Notes:

### Generate Runner key pair:
```
openssl genrsa 2048 | openssl pkcs8 -topk8 -nocrypt > private_key.pem
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

### Generate SAML Certificate:
```
openssl req -x509 -newkey rsa:2048 -keyout myservice.key -out myservice.cert -days 365 -nodes -subj "/CN=myservice.example.com"
```