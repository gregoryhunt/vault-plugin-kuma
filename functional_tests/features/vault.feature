Feature: Kuma User Tokens
  In order to test the Vault plugin for Kuma User Tokens
  I need to ensure the code funcionality is working as specified

  @kuma_roles
  Scenario: Configure and check roles
    Given I create the Vault role "kuma-role" with the following data
      ```
      {
        "dataplane_name": "backend-1",
        "mesh": "default",
        "ttl": "1h",
        "tags": "kuma.io/service=backend,kuma.io/service=backend-admin",
        "max_ttl": "24h"
      }
      ```
    Then I expect the role "kuma-role" to exist with the following data
      ```
      {
        "dataplane_name": "backend-1",
        "mesh": "default",
        "ttl": "1h0m0s",
        "max_ttl": "24h0m0s",
        "tags": "kuma.io/service=backend,kuma.io/service=backend-admin"
      }
      ```

  @kuma_dataplane_token
  Scenario: Create dataplane tokens
    Given I create the Vault role "kuma-dataplane-role" with the following data
      ```
      {
        "dataplane_name": "backend-1",
        "mesh": "default",
        "ttl": "1h",
        "tags": "kuma.io/service=backend,kuma.io/service=backend-admin",
        "max_ttl": "24h"
      }
      ```
    And I create a dataplane token for the role "kuma-dataplane-role"
    Then I should be able to start a dataplane using the token
    And a dataplane should be registered called "backend-1"



  #@kuma_user_token
  #Scenario: Create user tokens
  #  Given I create the Vault role "kuma-user-role" with the following data
  #    ```
  #    {
  #      "mesh": "default",
  #      "ttl": "1h",
  #      "groups": "mesh-system:admin"
  #      "max_ttl": "24h"
  #    }
  #    ```
  #  When I create a dataplane token for the role "kuma-role"
  #  Then I should be able to use this token to register the following dataplane
  #    ```
  #    {
  #      "type": "Dataplane",
  #      "name": "backend-1",
  #      "mesh": "default",
  #      "networking": {
  #        "address": "127.0.0.1",
  #        "inbound": [
  #          {
  #            "port": 11011,
  #            "servicePort": 11012,
  #            "tags": {
  #              "kuma.io/service": "backend",
  #              "version": "2.0",
  #              "env": "production"
  #            }
  #          }
  #        ],
  #        "outbound": [
  #          {
  #            "port": 33033,
  #            "service": "database"
  #          },
  #          {
  #            "port": 44044,
  #            "service": "user"
  #          }
  #        ]
  #      }
  #    }
  #    ```
