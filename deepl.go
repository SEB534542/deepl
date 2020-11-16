package deepl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	// V2 is the base url for v2 of the deepl API.
	V2 = "https://api.deepl.com/v2"
)

// New returns a usable deepl client.
func New(authKey string, opts ...ClientOption) *Client {
	c := Client{
		authKey: authKey,
		baseURL: V2,
	}

	for _, opt := range opts {
		opt(&c)
	}

	return &c
}

// A Client is a deepl client.
type Client struct {
	authKey string
	baseURL string
}

// A ClientOption configures the deepl client.
type ClientOption func(*Client)

// BaseURL sets the base url that is used for requests.
func BaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// Translate translates the given text into the given targetLang.
func (c *Client) Translate(ctx context.Context, text string, targetLang Language, opts ...TranslateOption) (string, Language, error) {
	translations, err := c.TranslateMany(ctx, []string{text}, targetLang, opts...)
	if err != nil {
		return "", "", fmt.Errorf("translate many: %w", err)
	}

	if len(translations) == 0 {
		return "", "", errors.New("deepl responded with no translations")
	}

	return translations[0].Text, Language(translations[0].DetectedSourceLanguage), nil
}

// A TranslateOption configures a translation.
type TranslateOption func(url.Values)

// SourceLang sets the source language of the text.
func SourceLang(lang Language) TranslateOption {
	return func(vals url.Values) {
		vals.Set("source_lang", string(lang))
	}
}

// SplitSentences configures the split_sentences option.
func SplitSentences(split SplitSentence) TranslateOption {
	return func(vals url.Values) {
		vals.Set("split_sentences", split.Value())
	}
}

// PreserveFormatting configures the preserve_formatting option.
func PreserveFormatting(preserve bool) TranslateOption {
	return func(vals url.Values) {
		vals.Set("preserve_formatting", boolString(preserve))
	}
}

// Formality configures the formality option.
func Formality(formal Formal) TranslateOption {
	return func(vals url.Values) {
		vals.Set("formality", formal.Value())
	}
}

// TranslateMany translates multiple texts into the given targetLang.
func (c *Client) TranslateMany(ctx context.Context, texts []string, targetLang Language, opts ...TranslateOption) ([]Translation, error) {
	vals := make(url.Values)
	vals.Set("auth_key", c.authKey)
	vals.Set("target_lang", string(targetLang))

	for _, text := range texts {
		vals.Add("text", text)
	}

	for _, opt := range opts {
		opt(vals)
	}

	resp, err := http.Post(c.translateURL(), "application/x-www-form-urlencoded", strings.NewReader(vals.Encode()))
	if err != nil {
		return nil, fmt.Errorf("deepl translate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, Error(resp.StatusCode)
	}

	var response translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode deepl response: %w", err)
	}

	return response.Translations, nil
}

func (c *Client) translateURL() string {
	return fmt.Sprintf("%s/translate", c.baseURL)
}

func boolString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Error is a deepl error.
type Error int

func (err Error) Error() string {
	switch err {
	case 456:
		return "Quota exceeded. The character limit has been reached."
	default:
		return http.StatusText(int(err))
	}
}