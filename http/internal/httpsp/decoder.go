package httpsp

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/MrMiaoMIMI/goshared/http/httpspi"
)

var _ httpspi.ResponseDecoder = (*jsonDecoder)(nil)

// jsonDecoder is the default ResponseDecoder that decodes JSON response bodies.
type jsonDecoder struct{}

func (d *jsonDecoder) Decode(resp *http.Response, v any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}
