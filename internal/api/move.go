package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tidwall/gjson"
)

// Move moves files or folders to a different parent folder.
func (p *PikPak) Move(fileIDs []string, parentID string) error {
	if len(fileIDs) == 0 {
		return errors.New("at least one file id is required")
	}

	body, err := json.Marshal(map[string]interface{}{
		"ids": fileIDs,
		"to":  map[string]string{"parent_id": parentID},
	})
	if err != nil {
		return err
	}

	const maxCaptchaRefreshes = 1
	for captchaRefreshes := 0; ; {
		req, err := p.newRequest("POST", "https://api-drive.mypikpak.com/drive/v1/files:batchMove", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Captcha-Token", p.CaptchaToken)
		req.Header.Set("X-Device-Id", p.DeviceId)

		response, statusCode, err := p.sendRequestWithStatus(req)
		if err != nil {
			return err
		}
		if !json.Valid(response) || !gjson.ParseBytes(response).IsObject() {
			return invalidMoveResponseError(statusCode)
		}

		errorCode := gjson.GetBytes(response, "error_code").Int()
		errorMessage := gjson.GetBytes(response, "error").String()
		if errorCode == 9 {
			if captchaRefreshes >= maxCaptchaRefreshes {
				return fmt.Errorf("move files failed: captcha challenge persisted after %d refresh", maxCaptchaRefreshes)
			}
			if err := p.AuthCaptchaToken("POST:/drive/v1/files:batchMove"); err != nil {
				return fmt.Errorf("refresh move captcha token: %w", err)
			}
			captchaRefreshes++
			continue
		}
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			if errorMessage != "" {
				return fmt.Errorf("move files failed: HTTP %d %s: %s", statusCode, http.StatusText(statusCode), errorMessage)
			}
			return fmt.Errorf("move files failed: HTTP %d %s", statusCode, http.StatusText(statusCode))
		}
		if errorCode != 0 {
			if errorMessage == "" {
				errorMessage = fmt.Sprintf("error code %d", errorCode)
			}
			return fmt.Errorf("move files failed: %s", errorMessage)
		}
		return nil
	}
}

func invalidMoveResponseError(statusCode int) error {
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("move files failed: HTTP %d %s returned a non-JSON response", statusCode, http.StatusText(statusCode))
	}
	return errors.New("move files failed: server returned an invalid JSON response")
}
