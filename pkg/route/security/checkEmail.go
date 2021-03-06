package security

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
	"gorm.io/gorm/clause"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/mail"
	"treehollow-v3-backend/pkg/route/contents"
	"treehollow-v3-backend/pkg/utils"
)

func checkEmailParamsCheckMiddleware(c *gin.Context) {
	recaptchaVersion := c.PostForm("recaptcha_version")
	recaptchaToken := c.PostForm("recaptcha_token")
	oldToken := c.PostForm("old_token")
	email := strings.ToLower(c.PostForm("email"))

	if len(email) > 100 || len(oldToken) > 32 || len(recaptchaToken) > 2000 || len(recaptchaVersion) > 2 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("CheckEmailParamsOutOfBound", "Wrong Parameter", logger.WARN))
		return
	}
	emailHash := utils.HashEmail(email)
	c.Set("email_hash", emailHash)
	c.Next()
}

func checkEmailRegexMiddleware(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))
	emailCheck, err := regexp.Compile(viper.GetString("email_check_regex"))
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "RegexError", "Server Error"))
		return
	}
	if !emailCheck.MatchString(email) {
		emailWhitelist := viper.GetStringSlice("email_whitelist")
		if _, ok := utils.ContainsString(emailWhitelist, email); !ok {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("EmailRegexCheckNotPass", "Sorry, we are only open to CMU community for now", logger.INFO))
			return
		}
	}
}

func checkEmailIsRegisteredUserMiddleware(c *gin.Context) {
	emailHash := c.MustGet("email_hash").(string)
	var count int64
	//check if user is registered
	err := base.GetDb(false).Where("email_hash = ?", emailHash).Model(&base.Email{}).Count(&count).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "SearchEmailHashFailed", consts.DatabaseReadFailedString))
		return
	}
	if count == 1 {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
		})
		c.Abort()
		return
	}
	c.Next()
}

//compatibility settings
func checkEmailIsOldTreeholeUserMiddleware(c *gin.Context) {
	oldToken := c.PostForm("old_token")
	emailHash := c.MustGet("email_hash").(string)
	var count int64

	//check if user is old v2 version user
	err := base.GetDb(false).Where("old_email_hash = ? and old_token = ?", emailHash, oldToken).
		Model(&base.User{}).Count(&count).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "SearchOldEmailHashFailed", consts.DatabaseReadFailedString))
		return
	}
	if count == 1 {
		c.JSON(http.StatusOK, gin.H{
			"code": 2,
		})
		c.Abort()
		return
	}
	c.Next()
}

func checkEmailRateLimitVerificationCode(c *gin.Context) {
	emailHash := c.MustGet("email_hash").(string)

	now := utils.GetTimeStamp()
	_, timeStamp, _, _ := base.GetVerificationCode(emailHash)
	if now-timeStamp < 60 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("TooMuchEmailInOneMinute", "Please wait 1 minute.", logger.INFO))
		return
	}
	c.Next()
}

func checkEmailReCaptchaValidationMiddleware(c *gin.Context) {
	recaptchaVersion := c.PostForm("recaptcha_version")
	recaptchaToken := c.PostForm("recaptcha_token")
	email := strings.ToLower(c.PostForm("email"))

	if len(c.PostForm("recaptcha_token")) < 1 {
		c.JSON(http.StatusOK, gin.H{
			"code": 3,
		})
		c.Abort()
		return
	}

	context, err2 := contents.EmailLimiter.Get(c, c.ClientIP())
	if err2 != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err2, "EmailLimiterFailed", consts.DatabaseReadFailedString))
		return
	}
	if context.Reached {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("EmailLimiterReached"+c.ClientIP(), "You sent too much code today. Please try again tomorrow.", logger.WARN))
		return
	}

	geoDb := utils.GeoDb.Get()
	if geoDb != nil && len(viper.GetStringSlice("allowed_register_countries")) != 0 {
		ip := net.ParseIP(c.ClientIP())
		record, err5 := geoDb.Country(ip)
		if err5 == nil {
			country := record.Country.Names["zh-CN"]
			if _, ok := utils.ContainsString(viper.GetStringSlice("allowed_register_countries"), country); !ok {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("RegisterNotAllowed"+c.ClientIP()+country+email, "Your country is not supported.", logger.WARN))
				return
			}
		}
	}

	var captcha recaptcha.ReCAPTCHA
	if recaptchaVersion == "v2" {
		captcha, _ = recaptcha.NewReCAPTCHA(viper.GetString("recaptcha_v2_private_key"), recaptcha.V2, 10*time.Second)
	} else {
		captcha, _ = recaptcha.NewReCAPTCHA(viper.GetString("recaptcha_v3_private_key"), recaptcha.V3, 10*time.Second)
	}
	captcha.ReCAPTCHALink = "https://www.recaptcha.net/recaptcha/api/siteverify"
	err := captcha.VerifyWithOptions(recaptchaToken, recaptcha.VerifyOption{
		RemoteIP:  c.ClientIP(),
		Threshold: float32(viper.GetFloat64("recaptcha_threshold")),
	})
	if err != nil {
		log.Println("recaptcha server error", err, c.ClientIP(), email)
		c.JSON(http.StatusOK, gin.H{
			"code": 3,
		})
		c.Abort()
		return
	}
	c.Next()
}

func checkEmail(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))

	emailHash := c.MustGet("email_hash").(string)

	code := utils.GenCode()

	err := mail.SendValidationEmail(code, email)
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SendEmailFailed"+email, "Failed to send code."))
		return
	}

	err = base.GetDb(false).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&base.VerificationCode{Code: code, EmailHash: emailHash, FailedTimes: 0, UpdatedAt: time.Now()}).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SaveVerificationCodeFailed", consts.DatabaseWriteFailedString))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  "Code sent successfully. Please check spams and wait 1 minute to resend.",
	})
}

func checkEmailInvitation(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))

	code := viper.GetString("invitation_code")

	err := mail.SendValidationEmail(code, email)
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SendEmailFailed"+email, "Failed to send code."))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 2,
		"msg":  "Code sent successfully. Please check spams and wait 1 minute to resend.",
	})
}

func unregisterEmail(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))

	emailHash := c.MustGet("email_hash").(string)

	code := utils.GenCode()

	err := mail.SendUnregisterValidationEmail(code, email)
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SendEmailFailed"+email, "Failed to send verification code."))
		return
	}

	err = base.GetDb(false).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&base.VerificationCode{Code: code, EmailHash: emailHash, FailedTimes: 0, UpdatedAt: time.Now()}).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SaveVerificationCodeFailed", consts.DatabaseWriteFailedString))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  "Verification code sent successfully.",
	})
}
