package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/hashicorp/go-hclog"
)

var opts = &godog.Options{
	Format: "pretty",
	Output: colors.Colored(os.Stdout),
}

var logStore bytes.Buffer
var logger hclog.Logger

var environment map[string]string

var createEnvironment = flag.Bool("create-environment", true, "Create and destroy the test environment when running tests?")
var alwaysLog = flag.Bool("always-log", false, "Always show the log output")
var dontDestroy = flag.Bool("dont-destroy", false, "Do not destroy the environment after the scenario")

func main() {
	godog.BindFlags("godog.", flag.CommandLine, opts)
	flag.Parse()

	status := godog.TestSuite{
		Name:                 "Kuma Vault Plugin Functional Tests",
		ScenarioInitializer:  initializeScenario,
		TestSuiteInitializer: initializeSuite,
		Options:              opts,
	}.Run()

	os.Exit(status)
}

func initializeSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		environment = map[string]string{}

		if *alwaysLog {
			logger = hclog.New(&hclog.LoggerOptions{Name: "functional-tests", Level: hclog.Trace, Color: hclog.AutoColor})

			logger.Info("Create standard logger")
		} else {

			logStore = *bytes.NewBufferString("")
			logger = hclog.New(&hclog.LoggerOptions{Output: &logStore, Level: hclog.Trace})
		}

		if *createEnvironment {

			cmd := exec.Command("/usr/local/bin/shipyard", "run", "--no-browser", "./shipyard")
			cmd.Dir = "../"
			cmd.Stderr = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error})
			cmd.Stdout = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug})

			err := cmd.Run()
			if err != nil {
				outputLog()
				os.Exit(1)
			}

		}

		environment["VAULT_ADDR"] = "http://localhost:8200"
		environment["VAULT_TOKEN"] = "root"

		tokenLoc := fmt.Sprintf("%s/.shipyard/data/kuma_config/admin.token", os.Getenv("HOME"))
		d, err := ioutil.ReadFile(tokenLoc)
		if err != nil {
			logger.Error("unable to read boostrap token", "file", tokenLoc)
			outputLog()
			os.Exit(1)
		}

		environment["KUMA_TOKEN"] = string(d)

		if environment["KUMA_TOKEN"] == "" {
			logger.Error("unable to fetch Kuma bootstrap token")
			outputLog()
			os.Exit(1)
		}

		configurePlugin()
	})

	ctx.ScenarioContext().After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if err != nil {
			outputLog()
		}

		return ctx, nil
	})

	ctx.AfterSuite(func() {
		if !*dontDestroy {
			cmd := exec.Command("shipyard", "destroy")
			cmd.Stderr = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error})
			cmd.Stdout = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Info})

			err := cmd.Run()
			if err != nil {
				outputLog()
				os.Exit(1)
			}
		}
	})
}

func outputLog() {
	if *alwaysLog {
		return
	}

	fmt.Printf("%s\n", string(logStore.Bytes()))
}

func initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^I create the Vault role "([^"]*)" with the following data$`, iCreateTheVaultRoleWithTheFollowingData)
	ctx.Step(`^I expect the role "([^"]*)" to exist with the following data$`, iExpectTheRoleToExistWithTheFollowingData)

	ctx.Step(`^I create a dataplane token for the role "([^"]*)"$`, iCreateADataplaneToken)
	ctx.Step(`^I should be able to use this token to register the following dataplane$`, iShouldBeAbleToUseThisTokenToRegisterTheFollowingDataplane)
}

func doVaultRequest(url, method, body string) (*http.Response, error) {
	req, err := http.NewRequest(
		method,
		fmt.Sprintf("%s/v1/%s", environment["VAULT_ADDR"], url),
		bytes.NewReader([]byte(body)))

	req.Header.Add("X-Vault-Token", environment["VAULT_TOKEN"])

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func iCreateTheVaultRoleWithTheFollowingData(arg1 string, arg2 *godog.DocString) error {
	resp, err := doVaultRequest("kuma/roles/"+arg1, http.MethodPost, arg2.Content)
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Expected status code 200 when creating Vault role, got %d, err: %s", resp.StatusCode, string(body))
	}

	return nil
}

func iExpectTheRoleToExistWithTheFollowingData(arg1 string, arg2 *godog.DocString) error {
	resp, err := doVaultRequest("kuma/roles/"+arg1, http.MethodGet, "")
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	j := map[string]interface{}{}
	json.Unmarshal(body, &j)

	data := j["data"].(map[string]interface{})

	testData := map[string]interface{}{}
	json.Unmarshal([]byte(arg2.Content), &testData)

	for k, v := range data {
		if v != testData[k] {
			return fmt.Errorf("Expected data for %s value %v, got %v", k, v, testData[k])
		}
	}

	return nil
}

func theExampleEnvironmentIsRunning() error {

	return nil
}

func configurePlugin() error {
	req := `
    {
      "type": "vault-plugin-kuma"
    }
  `

	logger.Debug("configuring plugin", "req", req)

	resp, err := doVaultRequest("sys/mounts/kuma", http.MethodPost, req)
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Expected status code 204 when creating Vault config, got %d, err: %s", resp.StatusCode, string(body))
	}

	req = `{
    "url": "http://kuma-cp.container.shipyard.run:5681",
    "token": "` + environment["KUMA_TOKEN"] + `"
  }`

	resp, err = doVaultRequest("kuma/config", http.MethodPost, req)
	if err != nil {
		return err
	}

	body, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Expected status code 204 when creating Vault config, got %d, err: %s", resp.StatusCode, string(body))
	}

	return nil
}

var lastToken = ""

func iCreateADataplaneToken(arg1 string) error {
	resp, err := doVaultRequest("kuma/creds/"+arg1, http.MethodGet, "")
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	j := map[string]interface{}{}
	json.Unmarshal(body, &j)

	if data, ok := j["data"].(map[string]interface{}); ok {
		token, ok := data["token"].(string)
		if ok {
			lastToken = token
			return nil
		}
	}

	return fmt.Errorf("unable to decode token response %v", j)
}

func iShouldBeAbleToUseThisTokenToRegisterTheFollowingDataplane(arg1 *godog.DocString) error {
	return godog.ErrPending
}
