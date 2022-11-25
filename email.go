package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var sendInBlueApiKey = ""

func sendVerificationEmail(emailContent []byte) (statusCode int, err error) {
	client := &http.Client{

		Transport: &http.Transport{},
		Timeout:   40 * time.Second,
	}

	req, _ := http.NewRequest("POST", "https://api.sendinblue.com/v3/smtp/email", bytes.NewBuffer(emailContent))
	req.Header.Add("accept", "application/json")
	req.Header.Add("api-key", sendInBlueApiKey)
	req.Header.Add("content-type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err.Error())
		time.Sleep(17 * time.Millisecond)

		if strings.Contains(err.Error(), "Unreach") {
			return sendVerificationEmail(emailContent)
		}
		if strings.Contains(err.Error(), "EOF") {
			return sendVerificationEmail(emailContent)
		}
		if strings.Contains(err.Error(), "imeout") {
			return sendVerificationEmail(emailContent)
		}
		log.Println(err)
		return statusCode, err
	}
	fmt.Printf("Email Verify:  %d\n", resp.StatusCode)
	//response_Bytes_Text, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	//StringText := string(response_Bytes_Text)

	return resp.StatusCode, err
}

type userObjectEmail struct {
	Username     string `json:"username"`
	UserEmail    string `json:"userEmail"`
	EmailContent string `json:"emailContent"`
	Subject      string `json:"subject"`
}

type senderEmailInfo struct {
	SenderEmail string `json:"senderEmail"`
	SenderName  string `json:"senderName"`
}

type emailTemplateStruct struct {
	Sender struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"sender"`
	Subject         string          `json:"subject"`
	HtmlContent     string          `json:"htmlContent"`
	MessageVersions []lastPartEmail `json:"messageVersions"`
}
type messageVersionInner struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type lastPartEmail struct {
	To          []messageVersionInner `json:"to"`
	HtmlContent string                `json:"htmlContent,omitempty"`
	Subject     string                `json:"subject,omitempty"`
}

func generateEmailTemplate(item userObjectEmail, from senderEmailInfo) ([]byte, error) {
	msgVersionInner := messageVersionInner{Email: item.UserEmail, Name: item.Username}
	this := lastPartEmail{HtmlContent: item.EmailContent, Subject: item.Subject, To: []messageVersionInner{msgVersionInner}}
	jsEmailObject := emailTemplateStruct{Sender: struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}(struct {
		Email string
		Name  string
	}{Email: from.SenderEmail, Name: from.SenderName}),
		HtmlContent:     item.EmailContent,
		Subject:         item.Subject,
		MessageVersions: []lastPartEmail{this, {To: []messageVersionInner{msgVersionInner}}},
	}
	personJSON, err := json.Marshal(jsEmailObject)

	return personJSON, err

}

func loadEmail(fileName string) (emailContent string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		emailContent += scanner.Text() + "\n"
	}
	return emailContent
}

func loadEmailTemplates() {
	files, err := ioutil.ReadDir("emails/")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if !f.IsDir() {

			emailTemplates[f.Name()] = loadEmail("emails/" + f.Name())
		}
	}
}

type replace struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

func emailPersonalise(email string, task []replace) string {
	for _, t := range task {
		email = strings.Replace(email, t.Before, t.After, -1)
	}
	return email
}

var emailTemplates = map[string]string{}

type emailFields struct {
	EmailTemplate string `json:"emailTemplate"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Url           string `json:"url"`
}

func sendEmail(e emailFields) (success bool) {

	var attempts int
	var subject string

	task := []replace{}

	switch e.EmailTemplate {
	case "verify.html":
		task = append(task, replace{Before: "VERIFYURL", After: e.Url})
		task = append(task, replace{Before: "PLAYERNAME", After: e.Name})
		subject = "Verify Email"
		fmt.Println("Selected Verify Email")
	case "forgot.html":
		task = append(task, replace{Before: "VERIFYURL", After: e.Url})
		task = append(task, replace{Before: "PLAYERNAME", After: e.Name})
		subject = "Verify Email"
		fmt.Println("Selected Forgot Password Email")
	default:
		fmt.Println("The guess is wrong!")
	}

	from := senderEmailInfo{SenderEmail: "support@" + site, SenderName: hostname}
	i := userObjectEmail{EmailContent: emailPersonalise(emailTemplates[e.EmailTemplate], task), UserEmail: e.Email, Username: e.Name, Subject: subject}

	resp, err := generateEmailTemplate(i, from)

	if err == nil {
		statusCode, err2 := sendVerificationEmail(resp)
		if statusCode == 201 {
			fmt.Println("successfully sent email!")
			return true
		}
		for err2 != nil {
			attempts++
			if attempts > 10 {
				break
			}
			if statusCode == 201 {
				fmt.Println("successfully sent email!")
				return true
			}
			statusCode, err2 = sendVerificationEmail(resp)
		}
		if statusCode == 201 {
			fmt.Println("successfully sent email!")
			return true
		}
	}
	return false
}
