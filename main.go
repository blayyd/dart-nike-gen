package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/fatih/color"
	"github.com/hugolgst/rich-go/client"
	"github.com/panjf2000/ants"
	//"github.com/corpix/uarand"
)

var provider int

func main() {
	startUp()
	richPresence()

	var numberOfAccounts int
	var thread int

	fmt.Println("How many accounts would you like to create?")
	fmt.Scan(&numberOfAccounts)

	fmt.Println("What provider would you like to use? \n[1]sms-activate \n[2]sms.discount")
	fmt.Scan(&provider)

	fmt.Println("How many threads would you like to run? (1-10)")
	fmt.Scan(&thread)

	var wg sync.WaitGroup
	// set capacity of goroutine pool
	p, _ := ants.NewPool(thread, ants.WithExpiryDuration(5*time.Minute))
	defer p.Release()

	syncProcess := func() {
		process()
		wg.Done()
		time.Sleep(15 * time.Second)
	}
	// Submit tasks one by one
	for i := 1; i <= numberOfAccounts; i++ {
		wg.Add(1)
		_ = p.Submit(syncProcess)
	}
	wg.Wait()

}

func startUp() {
	const logo = `
▓█████▄  ▄▄▄       ██▀███  ▄▄▄█████▓    ██▓ ▒█████
▒██▀ ██▌▒████▄    ▓██ ▒ ██▒▓  ██▒ ▓▒   ▓██▒▒██▒  ██▒
░██   █▌▒██  ▀█▄  ▓██ ░▄█ ▒▒ ▓██░ ▒░   ▒██▒▒██░  ██▒
░▓█▄   ▌░██▄▄▄▄██ ▒██▀▀█▄  ░ ▓██▓ ░    ░██░▒██   ██░
░▒████▓  ▓█   ▓██▒░██▓ ▒██▒  ▒██▒ ░    ░██░░ ████▓▒░
▒▒▓  ▒  ▒▒   ▓▒█░░ ▒▓ ░▒▓░  ▒ ░░      ░▓  ░ ▒░▒░▒░
░ ▒  ▒   ▒   ▒▒ ░  ░▒ ░ ▒░    ░        ▒ ░  ░ ▒ ▒░
░ ░  ░   ░   ▒     ░░   ░   ░          ▒ ░░ ░ ░ ▒
 ░          ░  ░   ░                  ░      ░ ░
░
`
	color.Cyan(logo)
	color.Magenta("	Created by blayyd#6969")
	color.Magenta("		in loving memory of dark")
}

func process() {

	type Config struct {
		Catchall  string `json:"catchall"`
		Webhook   string `json:"webhook"`
		Providers struct {
			SmsActivate struct {
				APIKey        string `json:"api-key"`
				CountryCode   string `json:"country-code"`
				CountryPrefix string `json:"country-prefix"`
				Country       string `json:"country"`
			} `json:"sms-activate"`
			SmsDiscount struct {
				APIKey        string `json:"api-key"`
				CountryPrefix string `json:"country-prefix"`
				Country       string `json:"country"`
			} `json:"sms-discount"`
		} `json:"providers"`
	}

	type Active struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Numbers struct {
			BookedOn    string `json:"booked_on"`
			CountryCode string `json:"country_code"`
			ID          string `json:"id"`
			Message     string `json:"message"`
			Number      string `json:"number"`
			Service     string `json:"service"`
		} `json:"numbers"`
	}

	type Order struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		ID      string `json:"id"`
		Number  string `json:"number"`
	}

	type Proxy struct {
		IP   string
		Port string
		User string
		Pass string
	}

	// global variables
	var smsActivateID string
	var robID string

	loadConfiguration := func(file string) Config {
		var config Config
		configFile, err := os.Open(file)
		defer configFile.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
		jsonParser := json.NewDecoder(configFile)
		jsonParser.Decode(&config)
		return config
	}

	var config Config = loadConfiguration("config.json")

	stringToProxy := func(line string) (Proxy, error) {

		parts := strings.Split(line, ":")

		if len(parts) == 2 { //ip:port format
			return Proxy{parts[0], parts[1], "", ""}, nil

		} else if len(parts) == 4 { //ip:port:user:pass format
			return Proxy{parts[0], parts[1], parts[2], parts[3]}, nil

		} else { //unknown format, error is returned
			return Proxy{"", "", "", ""}, errors.New("Error parsing proxy")
		}
	}

	loadProxy := func(filePath string) ([]Proxy, error) {
		file, err := os.Open(filePath)

		if err != nil {
			return nil, err
		}

		defer file.Close()

		var proxies []Proxy

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {

			proxy, err := stringToProxy(scanner.Text())

			if err == nil {
				proxies = append(proxies, proxy)
			}
		}

		return proxies, nil
	}

	getSmsActivate := func(id string) string {

		req, err := http.Get("https://sms-activate.ru/stubs/handler_api.php?api_key=" + config.Providers.SmsActivate.APIKey + "&action=getFullSms&id=" + id)

		if err != nil {
			log.Fatalln(err)
		}
		defer req.Body.Close()

		reqBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}
		reqString := string(reqBytes)

		return reqString
	}

	confirmSmsActivate := func(id string) {

		req, err := http.Get("https://sms-activate.ru/stubs/handler_api.php?api_key=" + config.Providers.SmsActivate.APIKey + "&action=setStatus&status=6&id=" + id)

		if err != nil {
			log.Fatalln(err)
		}
		defer req.Body.Close()
	}

	cancelSmsActivate := func(id string) {

		req, err := http.Get("https://sms-activate.ru/stubs/handler_api.php?api_key=" + config.Providers.SmsActivate.APIKey + "&action=setStatus&status=8&id=" + id)

		if err != nil {
			log.Fatalln(err)
		}
		defer req.Body.Close()
	}

	orderSmsActivate := func() (string, string) {
		var code string

		req, err := http.Get("https://sms-activate.ru/stubs/handler_api.php?api_key=" + config.Providers.SmsActivate.APIKey + "&action=getNumber&service=ew&ref=1124276&country=" + config.Providers.SmsActivate.CountryCode)

		if err != nil {
			log.Fatalln(err)
		}
		defer req.Body.Close()

		reqBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}
		reqString := string(reqBytes)

		if strings.Contains(reqString, "ACCESS_NUMBER") {
			reqSplit := strings.Split(reqString, ":")
			smsActivateID = string(reqSplit[1])
			code = getSmsActivate(smsActivateID)
			reqString = strings.TrimPrefix(reqSplit[2], config.Providers.SmsActivate.CountryPrefix)

		} else if strings.Contains(reqString, "NO_NUMBERS") {
			fmt.Println("No numbers found.")

		} else if strings.Contains(reqString, "NO_BALANCE") {
			fmt.Println("Insufficient balance.")
		}
		return reqString, code
	}

	orderRob := func() string {
		var order Order

		var orderJSON = []byte(`{"id": "37", "country_code": "` + config.Providers.SmsDiscount.CountryPrefix + `"}`)

		req, err := http.NewRequest("POST", "https://sms.discount/api/numbers/order", bytes.NewBuffer(orderJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("api-key", config.Providers.SmsDiscount.APIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()

		orderBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		orderString := string(orderBytes)

		json.Unmarshal([]byte(orderString), &order)
		robID = order.ID
		return order.Number
	}

	codeRob := func() string {
		var code Active

		req, err := http.NewRequest("GET", "https://sms.discount/api/numbers/active", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("api-key", config.Providers.SmsDiscount.APIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()

		codeBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		codeString := string(codeBytes)

		json.Unmarshal([]byte(codeString), &code)
		return code.Numbers.Message
	}

	archiveRob := func() {

		req, err := http.NewRequest("GET", "https://sms.discount/api/numbers/"+robID+"/archive", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("api-key", config.Providers.SmsDiscount.APIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()
	}

	successHook := func(email, password string) {
		webhookJSON := []byte(`{
			"content": null,
			"embeds": [
				{
					"title": "Successfully generated Nike account!",
					"color": 5814783,
					"fields": [
						{
							"name": "Account Email",
							"value": "||` + email + `||",
							"inline": true
						},
						{
							"name": "Account Password",
							"value": "||` + password + `||",
							"inline": true
						},
						{
							"name": "User:Pass Format",
							"value": "||` + email + `:` + password + `||"
						}
					],
					"footer": {
						"text": "made with pain and suffering by blayyd#6969",
						"icon_url": "https://pbs.twimg.com/profile_images/1366209650534748160/NsWZ_jrG_400x400.jpg"
					}
				}
			],
			"username": "Dart.IO",
			"avatar_url": "https://pbs.twimg.com/profile_images/1299529041070845953/4DcGZFsQ_400x400.jpg"
		}`)

		req, err := http.Post(config.Webhook, "application/json", bytes.NewBuffer(webhookJSON))
		if err != nil {
			log.Fatalln(err)
		}
		defer req.Body.Close()
	}

	registerAccount := func() {
		rand.Seed(time.Now().Unix())

		// input variables
		emailNumber := strconv.Itoa(randomdata.Number(1000000, 9999999))
		firstName := randomdata.FirstName(randomdata.RandomGender)
		lastName := randomdata.LastName()
		email := randomdata.FirstName(randomdata.RandomGender) + randomdata.LastName() + emailNumber + config.Catchall
		month := strconv.Itoa(randomdata.Number(10, 12))
		date := strconv.Itoa(randomdata.Number(10, 28))
		year := strconv.Itoa(randomdata.Number(1970, 1999))

		// password configuration
		digits := "0123456789"
		specials := "~=+%^*/()[]{}/!@#$?|"
		all := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"abcdefghijklmnopqrstuvwxyz" +
			digits + specials
		length := 20
		buf := make([]byte, length)
		buf[0] = digits[rand.Intn(len(digits))]
		buf[1] = specials[rand.Intn(len(specials))]
		for i := 2; i < length; i++ {
			buf[i] = all[rand.Intn(len(all))]
		}
		rand.Shuffle(len(buf), func(i, j int) {
			buf[i], buf[j] = buf[j], buf[i]
		})
		password := string(buf)

		// select random proxy
		proxies, err := loadProxy("proxies.txt")
		if err != nil {
			log.Fatalln(err)
		}
		var randomProxy Proxy
		randomProxy = proxies[rand.Intn(len(proxies))]

		// create chrome instance
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			// chromedp.DisableGPU,
			chromedp.ProxyServer("http://"+randomProxy.IP+":"+randomProxy.Port),
			// Set the headless flag to false to display the browser window
			chromedp.Flag("headless", false),
			chromedp.Flag("blink-settings", "imagesEnabled=false"),
			chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36"),
		)

		ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
		ctx, cancel = chromedp.NewContext(ctx)
		ctx, cancel = context.WithTimeout(ctx, 150*time.Second)
		defer cancel()

		chromedp.ListenTarget(ctx, func(ev interface{}) {
			go func() {
				switch ev := ev.(type) {
				case *fetch.EventAuthRequired:
					c := chromedp.FromContext(ctx)
					execCtx := cdp.WithExecutor(ctx, c.Target)

					resp := &fetch.AuthChallengeResponse{
						Response: fetch.AuthChallengeResponseResponseProvideCredentials,
						Username: randomProxy.User,
						Password: randomProxy.Pass,
					}

					err := fetch.ContinueWithAuth(ev.RequestID, resp).Do(execCtx)
					if err != nil {
						log.Print(err)
					}

				case *fetch.EventRequestPaused:
					c := chromedp.FromContext(ctx)
					execCtx := cdp.WithExecutor(ctx, c.Target)
					err := fetch.ContinueRequest(ev.RequestID).Do(execCtx)
					if err != nil {
						log.Print(err)
					}
				}
			}()
		})

		// variable randomizer
		genderRandomizer := rand.Intn(100)
		var genderPath string
		if genderRandomizer >= 50 {
			genderPath = `html/body/div[2]/div[3]/div[5]/form/div[7]/ul/li[1]/span`
		} else {
			genderPath = `/html/body/div[2]/div[3]/div[5]/form/div[7]/ul/li[2]/span`
		}

		// registers nike account up to phone number verification
		err = chromedp.Run(ctx,
			fetch.Enable().WithHandleAuthRequests(true),
			chromedp.Emulate(device.IPhone11Pro),
			chromedp.Navigate(`https://www.nike.com/register`))

		err = chromedp.Run(ctx,
			chromedp.WaitVisible(`/html/body/div[2]/div[3]/div[5]/form/div[9]/input`),
			chromedp.SendKeys(`/html/body/div[2]/div[3]/div[5]/form/div[1]/input`, email),
			chromedp.SendKeys(`/html/body/div[2]/div[3]/div[5]/form/div[2]/input`, password),
			chromedp.SendKeys(`/html/body/div[2]/div[3]/div[5]/form/div[3]/input`, firstName),
			chromedp.SendKeys(`/html/body/div[2]/div[3]/div[5]/form/div[4]/input`, lastName),
			chromedp.SendKeys(`/html/body/div[2]/div[3]/div[5]/form/div[5]/input`, month+"/"+date+"/"+year),
			chromedp.Click(genderPath),
			chromedp.Click(`/html/body/div[2]/div[3]/div[5]/form/div[9]/input`),
			// chromedp.WaitNotVisible(`/html/body/div[1]/div[3]/header/div/div[1]/div[1]/a/svg`),
			chromedp.Sleep(10*time.Second),
			chromedp.Navigate(`https://www.nike.com/member/settings`),
			chromedp.Click(`/html/body/div[3]/div/div[3]/div[1]/div[1]/div/div[2]/div/span`),
			chromedp.WaitVisible(`/html/body/div[3]/div/div[3]/div[2]/div/div/form/div[2]/div[4]/div/div/div/div[2]/button`), // Add
			chromedp.Click(`/html/body/div[3]/div/div[3]/div[2]/div/div/form/div[2]/div[4]/div/div/div/div[2]/button`),       // Add
			chromedp.WaitVisible(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/div[1]/input`),          // Mobile Number
		)

		if err != nil {
			cancel()
		} else {

			// call api for number and code
			if provider == 1 {
				phoneNumber, phoneCode := orderSmsActivate()

				err = chromedp.Run(ctx,
					chromedp.SendKeys(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/div[1]/select`, config.Providers.SmsActivate.Country),
					chromedp.Sleep(1*time.Second),
					chromedp.SendKeys(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/div[1]/input`, phoneNumber),
					chromedp.Sleep(1*time.Second),
					chromedp.Click(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/input`), // Send Code
				)

				for attemptCount := 1; attemptCount <= 9; attemptCount++ {
					if strings.Contains(phoneCode, "FULL_SMS") {
						phoneCode = strings.TrimPrefix(phoneCode, "FULL_SMS:Your Nike verification code is: ")
						fmt.Println("Code found:", phoneCode)
						err = chromedp.Run(ctx,
							chromedp.SendKeys(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[2]/input`, phoneCode),
							chromedp.Click(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[2]`),
							chromedp.Click(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[3]/input`),
						)

						// mark number as success
						confirmSmsActivate(smsActivateID)

						// write account to file
						f, err := os.OpenFile("accounts.txt",
							os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if err != nil {
							log.Println(err)
						}
						defer f.Close()
						if _, err := f.WriteString(email + ":" + password + "\n"); err != nil {
							log.Println(err)
						}

						// sends webhook
						successHook(email, password)

						fmt.Println("Successfully registered Nike account!")
						break

					} else if attemptCount <= 8 && phoneCode == "STATUS_WAIT_CODE" {
						fmt.Println("Waiting 15 seconds. Attempt:", attemptCount)
						err = chromedp.Run(ctx, chromedp.Sleep(15*time.Second))
						phoneCode = getSmsActivate(smsActivateID)

					} else if attemptCount == 9 && phoneCode == "STATUS_WAIT_CODE" {
						cancelSmsActivate(smsActivateID)
						fmt.Println("SMS attempt limit reached. Phone number cancelled.")

					} else if phoneCode == "STATUS_CANCEL" {
						fmt.Println("Activation cancelled.")
					}
				}
			} else if provider == 2 {
				phoneNumber := orderRob()
				phoneCode := codeRob()

				err = chromedp.Run(ctx,
					chromedp.SendKeys(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/div[1]/select`, config.Providers.SmsDiscount.Country),
					chromedp.Sleep(1*time.Second),
					chromedp.SendKeys(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/div[1]/input`, phoneNumber),
					chromedp.Sleep(1*time.Second),
					chromedp.Click(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[1]/input`), // Send Code
				)

				for attemptCount := 1; attemptCount <= 9; attemptCount++ {
					if codeRob() != "" {
						phoneCode = strings.TrimPrefix(phoneCode, "Your Nike verification code is: ")
						fmt.Println("Code found:", phoneCode)
						err = chromedp.Run(ctx,
							chromedp.SendKeys(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[1]/div[2]/input`, phoneCode),
							chromedp.Click(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[2]`),
							chromedp.Click(`/html/body/div[1]/div[1]/div/div[1]/div/div[10]/form/div[3]/input`),
						)

						// sends number to archive
						archiveRob()

						// write account to file
						f, err := os.OpenFile("accounts.txt",
							os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
						if err != nil {
							log.Println(err)
						}
						defer f.Close()
						if _, err := f.WriteString(email + ":" + password + "\n"); err != nil {
							log.Println(err)
						}

						// sends webhook
						successHook(email, password)

						fmt.Println("Successfully registered Nike account!")
						break

					} else if attemptCount <= 8 && phoneCode == "" {
						fmt.Println("Waiting 15 seconds. Attempt:", attemptCount)
						err = chromedp.Run(ctx, chromedp.Sleep(15*time.Second))
						phoneCode = codeRob()

					} else if attemptCount == 9 && phoneCode == "" {
						archiveRob()
						fmt.Println("SMS attempt limit reached. Phone number cancelled.")

					} else if phoneCode == "No numbers available" {
						fmt.Println("No numbers available")
					}
				}
			}
			cancel()
		}
	}
	registerAccount()
}

func richPresence() {
	err := client.Login("815781079238574110")
	if err != nil {
	}

	now := time.Now()
	err = client.SetActivity(client.Activity{
		State:      "Cooking...",
		Details:    "v0.1",
		LargeImage: "logo",
		LargeText:  "Dart.IO",
		SmallImage: "blayyd",
		SmallText:  "developed by blayyd#6969",
		Timestamps: &client.Timestamps{
			Start: &now,
		},
	})

	if err != nil {
	}
}
