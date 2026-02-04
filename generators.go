package main

import (
	"fmt"
	"math/rand"

	"github.com/go-faker/faker/v4"
)

// Generator is a function that returns a random string.
type Generator func() string

// Generators is a map of generator functions that can be used for variable substitution.
var Generators = map[string]Generator{
	"int":                  randomInt,
	"phone":                randomPhoneNumber,
	"datetime":             randomDateTime,
	"lat":                  randomLat,
	"long":                 randomLong,
	"real_address":         randomRealAddress,
	"cc_number":            randomCCNumber,
	"cc_type":              randomCCType,
	"email":                randomEmail,
	"domain_name":          randomDomainName,
	"ipv4":                 randomIPV4,
	"ipv6":                 randomIPV6,
	"password":             randomPassword,
	"jwt":                  randomJWT,
	"phone_number":         randomPhoneNumber,
	"mac_address":          randomMacAddress,
	"url":                  randomURL,
	"username":             randomUsername,
	"toll_free_number":     randomTollFreeNumber,
	"e_164_phone_number":   randomE164PhoneNumber,
	"title_male":           randomTitleMale,
	"title_female":         randomTitleFemale,
	"first_name":           randomFirstName,
	"first_name_male":      randomFirstNameMale,
	"first_name_female":    randomFirstNameFemale,
	"last_name":            randomLastName,
	"name":                 randomName,
	"unix_time":            randomUnixTime,
	"date":                 randomDate,
	"time":                 randomTime,
	"month_name":           randomMonthName,
	"year":                 randomYear,
	"day_of_week":          randomDayOfWeek,
	"day_of_month":         randomDayOfMonth,
	"timestamp":            randomTimestamp,
	"century":              randomCentury,
	"timezone":             randomTimeZone,
	"time_period":          randomTimePeriod,
	"word":                 randomWord,
	"sentence":             randomSentence,
	"paragraph":            randomParagraph,
	"currency":             randomCurrency,
	"amount":               randomAmount,
	"amount_with_currency": randomAmountWithCurrency,
	"uuid_hyphenated":      randomUUIDHyphenated,
	"uuid_digit":           randomUUIDDigit,
}

func randomInt() string {
	return fmt.Sprintf("%d", rand.Intn(1000000))
}

func randomName() string {
	return faker.Name()
}

func randomEmail() string {
	return faker.Email()
}

func randomUsername() string {
	return faker.Username()
}

func randomPassword() string {
	return faker.Password()
}

func randomURL() string {
	return faker.URL()
}

func randomWord() string {
	return faker.Word()
}

func randomSentence() string {
	return faker.Sentence()
}

func randomPhoneNumber() string {
	return faker.Phonenumber()
}

func randomDateTime() string {
	return faker.TimeString()
}

func randomLat() string {
	return fmt.Sprintf("%f", faker.Latitude())
}

func randomLong() string {
	return fmt.Sprintf("%f", faker.Longitude())
}

func randomRealAddress() string {
	addr := faker.GetRealAddress()
	return fmt.Sprintf("%s, %s, %s %s", addr.Address, addr.City, addr.State, addr.PostalCode)
}

func randomCCNumber() string {
	return faker.CCNumber()
}

func randomCCType() string {
	return faker.CCType()
}

func randomDomainName() string {
	return faker.DomainName()
}

func randomIPV4() string {
	return faker.IPv4()
}

func randomIPV6() string {
	return faker.IPv6()
}

func randomJWT() string {
	return faker.Jwt()
}

func randomMacAddress() string {
	return faker.MacAddress()
}

func randomTollFreeNumber() string {
	return faker.TollFreePhoneNumber()
}

func randomE164PhoneNumber() string {
	return faker.E164PhoneNumber()
}

func randomTitleMale() string {
	return faker.TitleMale()
}

func randomTitleFemale() string {
	return faker.TitleFemale()
}

func randomFirstName() string {
	return faker.FirstName()
}

func randomFirstNameMale() string {
	return faker.FirstNameMale()
}

func randomFirstNameFemale() string {
	return faker.FirstNameFemale()
}

func randomLastName() string {
	return faker.LastName()
}

func randomUnixTime() string {
	return fmt.Sprintf("%d", faker.UnixTime())
}

func randomDate() string {
	return faker.Date()
}

func randomTime() string {
	return faker.TimeString()
}

func randomMonthName() string {
	return faker.MonthName()
}

func randomYear() string {
	return faker.YearString()
}

func randomDayOfWeek() string {
	return faker.DayOfWeek()
}

func randomDayOfMonth() string {
	return faker.DayOfMonth()
}

func randomTimestamp() string {
	return faker.Timestamp()
}

func randomCentury() string {
	return faker.Century()
}

func randomTimeZone() string {
	return faker.Timezone()
}

func randomTimePeriod() string {
	return faker.Timeperiod()
}

func randomParagraph() string {
	return faker.Paragraph()
}

func randomCurrency() string {
	return faker.Currency()
}

func randomAmount() string {
	return fmt.Sprintf("%d.%02d", rand.Intn(1000), rand.Intn(100))
}

func randomAmountWithCurrency() string {
	return faker.AmountWithCurrency()
}

func randomUUIDHyphenated() string {
	return faker.UUIDHyphenated()
}

func randomUUIDDigit() string {
	return faker.UUIDDigit()
}
