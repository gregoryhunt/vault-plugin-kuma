Feature: Kuma User Tokens
  In order to test the Vault plugin for Kuma User Tokens
  I need to ensure the code funcionality is working as specified

  @kuma_roles
  Scenario: Canary Deployment existing candidate succeeds
    Given the example environment is running
    And the plugin is enabled and configured
    When I create the Vault role "kuma-role" with the following data
      ```
      {
        "mesh": "default",
        "ttl": "1h",
        "tags": "kuma.io/service=backend,kuma.io/service=backend-admin",
        "max_ttl": "24h"
      }
      ```
    Then I expect the role "kuma-role" to exist with the following data
      ```
      {
        "mesh": "default",
        "ttl": 3600,
        "max_ttl": 86400,
        "tags": "kuma.io/service=backend,kuma.io/service=backend-admin"
      }
      ```
