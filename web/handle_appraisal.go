package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/evepraisal/go-evepraisal"
	"github.com/evepraisal/go-evepraisal/esi"
	"github.com/evepraisal/go-evepraisal/legacy"
	"github.com/evepraisal/go-evepraisal/discord"
	"github.com/go-zoo/bone"
	"github.com/dustin/go-humanize"
)

var (
	errInputTooBig = errors.New("Input value is too big")
	errInputEmpty  = errors.New("Input value is empty")
)

// AppraisalPage contains data used on the appraisal page
type AppraisalPage struct {
	Appraisal *evepraisal.Appraisal `json:"appraisal"`
	Status    *esi.ContractStatus   `json:"status"`
	ShowFull  bool                  `json:"show_full,omitempty"`
	IsOwner   bool                  `json:"is_owner,omitempty"`
}

func appraisalLink(appraisal *evepraisal.Appraisal) string {
	if appraisal.Private {
		return fmt.Sprintf("/a/%s/%s", appraisal.ID, appraisal.PrivateToken)
	}
	return fmt.Sprintf("/a/%s", appraisal.ID)
}

func parseAppraisalBody(r *http.Request) (string, error) {
	// Parse body
	r.ParseMultipartForm(20 * 1000)

	var body string
	f, _, err := r.FormFile("uploadappraisal")
	if err == http.ErrNotMultipart || err == http.ErrMissingFile {
		body = r.FormValue("raw_textarea")
	} else if err != nil {
		return "", err
	} else {
		defer f.Close()
		bodyBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return "", err
		}
		body = string(bodyBytes)
	}
	if len(body) > 200000 {
		return "", errInputTooBig
	}

	if len(body) == 0 {
		return "", errInputEmpty
	}
	return body, nil
}

// HandleAppraisal is the handler for POST /appraisal
func (ctx *Context) HandleAppraisal(w http.ResponseWriter, r *http.Request) {

	persist := r.FormValue("persist") != "no"

	body, err := parseAppraisalBody(r)
	if err != nil {
		ctx.renderErrorPage(r, w, http.StatusBadRequest, "Invalid input", err.Error())
		return
	}

	errorRoot := PageRoot{}
	errorRoot.UI.RawTextAreaDefault = body

	// Parse Market
	market := r.FormValue("market")

	// Legacy Market ID
	marketID, err := strconv.ParseInt(market, 10, 64)
	if err == nil {
		var ok bool
		market, ok = legacy.MarketIDToName[marketID]
		if !ok {
			ctx.renderErrorPage(r, w, http.StatusBadRequest, "Invalid input", "Market not found.")
			return
		}
	}

	// No market given
	if market == "" {
		ctx.renderErrorPageWithRoot(r, w, http.StatusBadRequest, "Invalid input", "A market is required.", errorRoot)
		return
	}

	// Invalid market given
	foundMarket := false
	for _, m := range selectableMarkets {
		if m.Name == market {
			foundMarket = true
			break
		}
	}
	if !foundMarket {
		ctx.renderErrorPageWithRoot(r, w, http.StatusBadRequest, "Invalid input", "Given market is not valid.", errorRoot)
		return
	}

	user := ctx.GetCurrentUser(r)

	buybackCap := 0.0
	if user != nil {
		affiliation, found := esi.NewOauthFetcher(ctx.App.TypeDB, ctx.OauthClient(r)).GetCharacterAffiliation(user.CharacterID)
		if !found {
			ctx.renderErrorPageWithRoot(r, w, http.StatusBadRequest, "Invalid character", "Unknown character.", errorRoot)
			return
		}

		if affiliation.AllianceID != esi.ValidAlliance {
			ctx.renderErrorPageWithRoot(r, w, http.StatusBadRequest, "Invalid character", "Not in TEST alliance.", errorRoot)
			return
		}

		buybackCap = evepraisal.BuybackCapTEST
		for _, corpID := range evepraisal.IPOrgCorporations {
			if corpID == affiliation.CorporationID {
				buybackCap = evepraisal.BuybackCapIPOrg
				break;
			}
		}
	}

	visibility := r.FormValue("visibility")
	private := false
	if visibility == "private" && user != nil {
		private = true
	}

	// Actually do the appraisal
	appraisal, err := ctx.App.StringToAppraisal(market, body)
	if err == evepraisal.ErrNoValidLinesFound {
		log.Println("No valid lines found:", spew.Sdump(body))
		ctx.renderErrorPageWithRoot(r, w, http.StatusBadRequest, "Invalid input", err.Error(), errorRoot)
		return
	} else if err != nil {
		ctx.renderErrorPageWithRoot(r, w, http.StatusBadRequest, "Invalid input", err.Error(), errorRoot)
		return
	}

	appraisal.BuybackCap = buybackCap
	appraisal.User = user
	appraisal.Private = private
	appraisal.PrivateToken = NewPrivateAppraisalToken()

	// Persist Appraisal to the database
	if persist {
		err = ctx.App.AppraisalDB.PutNewAppraisal(appraisal)
		if err != nil {
			ctx.renderServerErrorWithRoot(r, w, err, errorRoot)
			return
		}
	}

	// Log for later analyics
	log.Println(appraisal)

	// Set new session variable
	ctx.setSessionValue(r, w, "market", market)
	ctx.setSessionValue(r, w, "visibility", visibility)
	ctx.setSessionValue(r, w, "persist", persist)

	sort.Slice(appraisal.Original.Items, func(i, j int) bool {
		return appraisal.Original.Items[i].RepresentativePrice() > appraisal.Original.Items[j].RepresentativePrice()
	})

	var status *esi.ContractStatus = nil
	if user != nil && appraisal.OwnerID == user.CharacterID {
		status = esi.NewOauthFetcher(ctx.App.TypeDB, ctx.OauthClient(r)).GetContractStatus(user, appraisal)
	}

	// Render the new appraisal to the screen (there is no redirect here, we set the URL using javascript later)
	w.Header().Add("X-Appraisal-ID", appraisal.ID)
	ctx.render(r, w, "appraisal.html",
		AppraisalPage{
			IsOwner:   IsAppraisalOwner(user, appraisal),
			Appraisal: cleanAppraisal(appraisal),
			Status: status,
		},
	)
}

// HandleViewAppraisal is the handler for /a/[id]
func (ctx *Context) HandleViewAppraisal(w http.ResponseWriter, r *http.Request) {
	// Legacy Logic
	appraisalID := bone.GetValue(r, "appraisalID")
	if bone.GetValue(r, "legacyAppraisalID") != "" {
		legacyAppraisalIDStr := bone.GetValue(r, "legacyAppraisalID")
		suffix := filepath.Ext(legacyAppraisalIDStr)
		legacyAppraisalIDStr = strings.TrimSuffix(legacyAppraisalIDStr, suffix)
		legacyAppraisalID, err := strconv.ParseUint(legacyAppraisalIDStr, 10, 64)
		if err != nil {
			ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
			return
		}
		appraisalID = evepraisal.Uint64ToAppraisalID(legacyAppraisalID) + suffix
	}

	appraisal, err := ctx.App.AppraisalDB.GetAppraisal(appraisalID)
	if err == evepraisal.AppraisalNotFound {
		ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
		return
	} else if err != nil {
		ctx.renderServerError(r, w, err)
		return
	}

	user := ctx.GetCurrentUser(r)
	isOwner := IsAppraisalOwner(user, appraisal)

	if appraisal.Private {
		correctToken := appraisal.PrivateToken == bone.GetValue(r, "privateToken")
		if !(isOwner || correctToken) {
			ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
			return
		}
	} else if bone.GetValue(r, "privateToken") != "" {
		ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
		return
	}

	appraisal = cleanAppraisal(appraisal)

	sort.Slice(appraisal.Original.Items, func(i, j int) bool {
		return appraisal.Original.Items[i].RepresentativePrice() > appraisal.Original.Items[j].RepresentativePrice()
	})

	if r.Header.Get("format") == "json" {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appraisal)
		return
	}

	if r.Header.Get("format") == "raw" {
		io.WriteString(w, appraisal.Raw)
		return
	}

	var status *esi.ContractStatus = nil
	if user != nil && appraisal.OwnerID == user.CharacterID {
		status = esi.NewOauthFetcher(ctx.App.TypeDB, ctx.OauthClient(r)).GetContractStatus(user, appraisal)
		state, _, found := ctx.App.AppraisalDB.GetNotifiedState(appraisalID)
		if!found || state != status.Summary {
			switch status.Summary {
			case "valid":
				discord.PostMessage(fmt.Sprintf("@bb Contract *%s* is VALID and ready for acceptance! Character: *%s* Amount: *%s*", status.Title, user.CharacterName, humanize.Commaf(appraisal.BuybackOffer())))
			case "invalid":
				discord.PostMessage(fmt.Sprintf("@bb Contract *%s* is INVALID and should be rejected!", status.Title))
			case "deleted":
				if state == "invalid" {
					discord.PostMessage(fmt.Sprintf("@bb Contract *%s* was DELETED and can be forgotten!", status.Title))
				}
			}

			ctx.App.AppraisalDB.SetNotifiedState(appraisalID, status.Summary)
		}
	}

	ctx.render(r, w, "appraisal.html",
		AppraisalPage{
			Appraisal: appraisal,
			Status:    status,
			ShowFull:  r.FormValue("full") != "",
			IsOwner:   isOwner,
		})
}

// HandleDeleteAppraisal is the handler for POST /a/delete/[id]
func (ctx *Context) HandleDeleteAppraisal(w http.ResponseWriter, r *http.Request) {
	appraisalID := bone.GetValue(r, "appraisalID")
	appraisal, err := ctx.App.AppraisalDB.GetAppraisal(appraisalID)
	if err == evepraisal.AppraisalNotFound {
		ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
		return
	} else if err != nil {
		ctx.renderServerError(r, w, err)
		return
	}

	if !IsAppraisalOwner(ctx.GetCurrentUser(r), appraisal) {
		ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
		return
	}

	err = ctx.App.AppraisalDB.DeleteAppraisal(appraisalID)
	if err == evepraisal.AppraisalNotFound {
		ctx.renderErrorPage(r, w, http.StatusNotFound, "Not Found", "I couldn't find what you're looking for")
		return
	} else if err != nil {
		ctx.renderServerError(r, w, err)
		return
	}

	ctx.setFlashMessage(r, w, FlashMessage{Message: fmt.Sprintf("Appraisal %s has been deleted.", appraisalID), Severity: "success"})
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// NewPrivateAppraisalToken returns a new token to use for private appraisals
func NewPrivateAppraisalToken() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 16)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
