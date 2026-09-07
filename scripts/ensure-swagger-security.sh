#!/usr/bin/env bash

set -euo pipefail

swagger_dir="services/gateway/docs"
json_file="${swagger_dir}/swagger.json"
yaml_file="${swagger_dir}/swagger.yaml"

# swag поддерживает security definitions, но не добавляет глобальный
# security requirement из блока общей информации Swagger 2.0.
perl -0pi -e 's/\n}\s*\z/,\n    "security": [\n        {\n            "BearerAuth": []\n        }\n    ]\n}\n/' "${json_file}"
sed -i '/^securityDefinitions:/i security:\n- BearerAuth: []' "${yaml_file}"
