package mail

import (
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
	"strconv"
)

func SendValidationEmail(code string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("from_domain"))
	m.SetHeader("To", recipient)
	var title string
	if(len(code) > 6) {
		title = "[" + websiteName + "] Invitation Code"
	} else {
		title = "[" + websiteName + "] Validation Code"
	}
	m.SetHeader("Subject", title)

	msg := `<!DOCTYPE html>
<html lang="cn">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
</head>
<body>
<p>Welcome to ` + websiteName + `!</p>
<p>This is your verification code. It is valid for 12 hours.</p>
<p><strong>` + code + `</strong></p>
</body>
</html>`

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	m.SetBody("text/html", msg)
	m.AddAlternative("text/plain", "Hi,\n\nWelcome to "+websiteName+"!\n\n"+code+"\nThis is your verification code. It is valid for 12 hours.\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err = d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func SendUnregisterValidationEmail(code string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("from_domain"))
	m.SetHeader("To", recipient)
	title := "[" + websiteName + "] Verification Code"
	m.SetHeader("Subject", title)

	msg := `<!DOCTYPE html>
<html lang="cn">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
</head>
<body>
<p>Hi, You are deleting your account on ` + websiteName + `。</p>
<p>This is your verification code. It is valid for 12 hours.</p>
<p><strong>` + code + `</strong></p>
</body>
</html>`

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	m.SetBody("text/html", msg)
	m.AddAlternative("text/plain", "Hi,\n\nYou are deleting your account on "+websiteName+"。\n\n"+code+"\nThis is your verification code. It is valid for 12 hours.\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err = d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func SendPasswordNonceEmail(nonce string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("smtp_username"))
	m.SetHeader("To", recipient)
	title := "Welcome to " + websiteName
	m.SetHeader("Subject", title)

	msg := `<!DOCTYPE html>
<html lang="cn">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
</head>
<body>
<p>Welcome to ` + websiteName + `!</p>
<p>The string below is necessary to delete your account. Please keep it safe.</p>
<p><strong>` + nonce + `</strong></p>
</body>
</html>`

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	m.SetBody("text/html", msg)
	m.AddAlternative("text/plain", "Hi,\n\nWelcome to "+websiteName+"!\nThe string below is necessary to delete your account. Please keep it safe.\n"+nonce+"\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err = d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
