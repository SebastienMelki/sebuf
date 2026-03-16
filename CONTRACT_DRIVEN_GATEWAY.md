# Krakend endpoint template spec for sebuf

This doc describes **exactly what the `sebuf` generator should emit** in KrakenD Flexible Config templates, given high-level endpoint semantics from proto. It is meant to be consumed by the `sebuf` implementation.
---

## What the generator needs from proto (high level)

At a minimum, for each endpoint the generator must be able to infer:

- Path, HTTP method and backend URL pattern.
- Header profile (which `*_input_headers.tmpl` or explicit headers to use).
- Whether JWT is required.
- Whether recaptcha protection is required.
- Whether a **custom timeout** is needed (otherwise fall back to the root service timeout from env/settings).

---

## What the generator must NOT own

- Service URLs / hosts (`*_host`), feature flags (`enable_*`), env-specific values in `settings/**/vars.json`.
- Telemetry, logging, CORS, router and HTTP server settings.
- The actual contents of partials/templates under `config/partials/*.tmpl` and `config/templates/*.tmpl` (other than the generated endpoint fragments).

---

## What the generator should emit (building blocks)

When generating endpoint fragments (`*.tmpl` under `config/templates/`), the generator should compose them from a small set of reusable pieces and respect Krakend’s JSON shape.

- **Header partials**
  - Examples:
    - `{{ include "trading_input_headers.tmpl" }}`  
    - `{{ include "mainapp_input_headers.tmpl" }}`  
    - `{{ include "default_input_headers.tmpl" }}`
  - The generator can **either**:
    - Emit explicit `"input_headers": [...]` arrays when it knows the exact header list, **or**
    - Use these partials and let the gateway team evolve header sets centrally.

- **JWT validator template**
  - For endpoints that require JWT:
    ```tmpl
    "extra_config": {
      {{ template "jwt_auth_validator.tmpl" . }}
    }
    ```

- **Recaptcha (bot protection) partial**
  - For endpoints that must be protected with recaptcha:
    ```tmpl
    "extra_config": {
      {{ include "recpatcha_validator.tmpl" }},
      ...
    }
    ```
  - Combining both JWT and recaptcha:
    ```tmpl
    "extra_config": {
      {{ include "recpatcha_validator.tmpl" }},
      {{ template "jwt_auth_validator.tmpl" . }}
    }
    ```

### Endpoint timeout

- For endpoints that need a specific timeout, the generator can add:
  ```tmpl
  "timeout": "90s"
  ```
- If no per-endpoint timeout is emitted, Krakend will fall back to the **root service timeout** configured via env/settings (e.g. `.vars.timeout` in `settings/**/vars.json`).
### Notes about `extra_config` and JSON shape

- Each **endpoint** has its own top-level `"extra_config"` (sibling of `"backend"`).  
- Each **backend entry** can **also** have its own `"extra_config"` (sibling of `"host"`, `"method"`, etc.).  
- Both endpoint-level and backend-level `"extra_config"` can contain **multiple keys** (e.g. `"backend/http"`, `"modifier/lua-endpoint"`, `"qos/ratelimit/proxy"`):
  - The generator must merge all entries into **one object per level**.
  - Be careful with commas when emitting multiple entries inside the same `"extra_config"` block.

The generator’s job is to decide, per endpoint, **which of these pieces to include** based on proto semantics, and then render JSON-shaped fragments that the root `krakend.tmpl` can inject into the global `endpoints` array.

---

## Examples: portfolio, charts, and auth endpoint fragments

### `portfolio_endpoints.tmpl`

This is an example of the **desired output** for the portfolio service:

```tmpl
{
    "endpoint": "/api/v1/portfolio",
    "method":"POST",
    "output_encoding":"json",
    {{ include "trading_input_headers.tmpl" }},
    "backend":[
       {
          "url_pattern":"/api/v1/portfolio",
          "encoding":"json",
          "sd":"static",
          "method":"POST",
          "host": ["{{ .vars.portfolio_host }}" ],
          "disable_host_sanitize":false,
          "extra_config": {
             "backend/http": {
               "return_error_code": true
             }
           }         
       }
    ],  
    "extra_config":{      
        {{ template "jwt_auth_validator.tmpl" . }}
    }   
 },
 {
    "endpoint": "/api/v1/portfolio/latest",
    "method":"POST",
    "output_encoding":"json",
    {{ include "trading_input_headers.tmpl" }},
    "backend":[
       {
          "url_pattern":"/api/v1/portfolio/latest",
          "encoding":"json",
          "sd":"static",
          "method":"POST",
          "host": ["{{ .vars.portfolio_host }}" ],
          "disable_host_sanitize":false,
          "extra_config": {
             "backend/http": {
               "return_error_code": true
             }
           }         
       }
    ],  
    "extra_config":{      
        {{ template "jwt_auth_validator.tmpl" . }}
    }   
 }
```

Key points:

- Both endpoints:
  - Use the **same header profile** (`trading_input_headers.tmpl`).
  - Use the same host var (`.vars.portfolio_host`).
  - Use the same backend `extra_config.backend/http.return_error_code` to **propagate** backend HTTP error codes to the client. If instead you need to obfuscate backend errors, do **not** set `"return_error_code": true`.
  - Are **JWT-protected** via `jwt_auth_validator.tmpl`.
- This file is a **fragment** (no surrounding `[]`), intended to be inserted into the root `endpoints` array via:
  ```tmpl
  {{ template "portfolio_endpoints.tmpl" . }}
  ```

### `charts_endpoints.tmpl`

The charts endpoints are similar, but also demonstrate `input_query_strings`:

```tmpl
{
    "endpoint": "/api/v1/options/{contract}/charts",
    "method":"GET",
    "output_encoding":"json",
    {{ include "trading_input_headers.tmpl" }},
    "input_query_strings": [
        "period"
    ],
    "backend":[
       {
          "url_pattern":"/v1/options/{contract}/charts",
          "encoding":"json",
          "sd":"static",
          "method":"GET",
          "host": ["{{ .vars.charts_host }}" ],
          "disable_host_sanitize":false,
          "extra_config": {
             "backend/http": {
               "return_error_code": true
             }
           }         
       }
    ],
    "extra_config":{
        {{ template "jwt_auth_validator.tmpl" . }}
    }
 },
 {
    "endpoint": "/api/v1/assets/{symbol}/charts",
    "method":"GET",
    "output_encoding":"json",
    {{ include "trading_input_headers.tmpl" }},
    "input_query_strings": [
        "period"
    ],
    "backend":[
       {
          "url_pattern":"/v1/assets/{symbol}/charts",
          "encoding":"json",
          "sd":"static",
          "method":"GET",
          "host": ["{{ .vars.charts_host }}" ],
          "disable_host_sanitize":false,
          "extra_config": {
             "backend/http": {
               "return_error_code": true
             }
           }         
       }
    ],
    "extra_config":{
        {{ template "jwt_auth_validator.tmpl" . }}
    }
 }
```

Generator implications:

- For endpoints with **query parameters**, emit `"input_query_strings": [...]` to list the accepted query keys (here: `"period"`), and mirror them to the backend URL as needed.
- The rest of the structure (headers, `sd`, `encoding`, `disable_host_sanitize`, `backend/http.return_error_code`, JWT validator) follows the same rules as the portfolio example.

### `auth_endpoints.tmpl` (recaptcha + JSON-schema + header injection)

The first auth endpoint shows how to combine recaptcha protection, JSON-schema validation, and a backend header injection:

```tmpl
{
   "endpoint": "/api/authz/users/register",
   "method":"POST",
   "output_encoding":"json",
   {{ include "auth_input_headers.tmpl" }},
   "backend":[
      {
         "url_pattern":"/api/v2/users/register/",
         "encoding":"json",
         "sd":"static",
         "method":"POST",
         "host": ["{{ .vars.auth_host }}" ],
         "disable_host_sanitize":false,
         "extra_config": {
            "modifier/martian": {
               "header.Append": {
                 "scope": ["request"],
                 "name": "x-api-key",
                 "value": "{{ env "AUTH_API_KEY" }}"
               }
             },
            "backend/http": {
              "return_error_code": true
            }
          }         
      }
   ],  
   "extra_config":{      
      {{ include "recpatcha_validator.tmpl" }},
      "validation/json-schema":{
         "type":"object",
         "required":["email","com_first_name","com_last_name"],
         "properties":{
            "email":{
               "type":"string",
               "format": "email"
            },           
            "com_first_name":{
               "type":"string"
            },
            "com_last_name":{
               "type":"string"
            },
            "uae_pass_auth_code":{
               "type":"string"
            }
         }
      }
   }   
},
```

Generator implications:

- **Recaptcha protection**  
  - When an endpoint requires bot protection, include:
    ```tmpl
    {{ include "recpatcha_validator.tmpl" }},
    ```
    inside endpoint `"extra_config"`, before any other keys you need to add.

- **JSON-schema validation**  
  - When request-body validation is needed, emit a `"validation/json-schema"` block at endpoint level describing:
    - `"type"` (usually `"object"`),
    - `"required"` fields,
    - `"properties"` with types/formats.  
  - The generator does not need to derive the full schema from proto automatically today, but it should be **able to include** a provided schema block if one is available.

- **Backend header injection (low priority for generator)**  
  - The backend `"extra_config"` above uses the Martian modifier to append an `x-api-key` header before calling the auth backend:
    ```tmpl
    "extra_config": {
      "modifier/martian": {
         "header.Append": {
           "scope": ["request"],
           "name": "x-api-key",
           "value": "{{ env "AUTH_API_KEY" }}"
         }
       },
       "backend/http": {
         "return_error_code": true
       }
    }
    ```
  - Support for automatically generating such header injections from proto is **nice-to-have / low priority**. The generator must, however, preserve/merge any existing `"modifier/martian"` blocks if they are present in hand-written templates.

---

## Summary

- **Input**: per-endpoint semantics from proto (path, method, backend pattern, header profile / explicit headers, JWT yes/no, recaptcha yes/no, timeout requirements).
- **Output**: JSON-shaped `.tmpl` fragments composed from:
  - `{{ include "<profile>_input_headers.tmpl" }}` or explicit `"input_headers": [...]`  
  - `{{ template "jwt_auth_validator.tmpl" . }}` when JWT is needed  
  - `{{ include "recpatcha_validator.tmpl" }}` when recaptcha is needed  
- The root `krakend.tmpl` remains the single place that wires these fragments into the global config via `{{ template "<service>_endpoints.tmpl" . }}`.

**Default field recommendations:** unless there is a very specific reason not to, the generator should:

- Set backend `"sd": "static"` and `"encoding": "json"` (pure structural defaults; do **not** pull them from env files).
- Set endpoint `"output_encoding": "json"` by default for all generated HTTP endpoints.
- Set `"disable_host_sanitize": false` on backends (safer default; only relax if a backend explicitly requires preserving the `Host` header). This flag should also be treated as a code default, not read from env.
