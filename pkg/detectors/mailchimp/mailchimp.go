package mailchimp

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/trufflesecurity/trufflehog/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/pkg/pb/detectorspb"

	"github.com/trufflesecurity/trufflehog/pkg/common"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time
var _ detectors.Detector = (*Scanner)(nil)

var (
	// TODO: Other country patterns?
	keyPat = regexp.MustCompile(`[0-9a-f]{32}-us[0-9]{1,2}`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"-us"}
}

// FromData will find and optionally verify Mailchimp secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	//pretty standard regex match
	matches := keyPat.FindAllString(dataStr, -1)

	for _, match := range matches {

		s := detectors.Result{
			DetectorType: detectorspb.DetectorType_Mailchimp,
			Raw:          []byte(match),
		}

		if verify {
			datacenter := strings.Split(match, "-")[1]

			client := common.SaneHttpClient()
			// https://mailchimp.com/developer/guides/marketing-api-conventions/
			req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s.api.mailchimp.com/3.0/", datacenter), nil)
			req.SetBasicAuth("anystring", match)
			res, err := client.Do(req)
			if err != nil {
				break
			}
			defer res.Body.Close()
			if res.StatusCode == 200 {
				s.Verified = true
			} else {
				s.Verified = false
			}

		}

		if !s.Verified {
			if detectors.IsKnownFalsePositive(string(s.Raw), detectors.DefaultFalsePositives, true) {
				continue
			}
		}

		results = append(results, s)
	}

	return
}