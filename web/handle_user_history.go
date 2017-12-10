package web

import (
	"fmt"
	"net/http"

	"github.com/evepraisal/go-evepraisal"
	"github.com/evepraisal/go-evepraisal/esi"
)

type StatusedAppraisal struct {
	Appraisal evepraisal.Appraisal
	Status esi.ContractStatus
}

// HandleUserHistoryAppraisals is the handler for /user/latest
func (ctx *Context) HandleUserHistoryAppraisals(w http.ResponseWriter, r *http.Request) {
	user := ctx.GetCurrentUser(r)
	before := r.FormValue("before")
	limit := 50
	appraisals, err := ctx.App.AppraisalDB.LatestAppraisalsByUser(*user, limit+1, r.FormValue("kind"), before)
	if err != nil {
		ctx.renderServerError(r, w, err)
		return
	}

	hasMore := len(appraisals) > limit
	var next string
	if hasMore {
		next = fmt.Sprintf("/user/history?before=%s", appraisals[len(appraisals)-2].ID)
	}

	cleanAppraisals := cleanAppraisals(appraisals)
	// Cut off the extra one appraisal that we may have gotten (to see if there's another page)
	if len(appraisals) > limit {
		cleanAppraisals = cleanAppraisals[0:limit]
	}

	var history = make([]StatusedAppraisal,len(cleanAppraisals))
	if len(history) > 0 {
		cf := esi.NewContractFetcher(ctx.App.TypeDB, ctx.OauthToken(r))
		contracts, _ := cf.GetContracts(user.CharacterID)
		for index, appraisal := range cleanAppraisals {
			history[index].Appraisal = appraisal
			history[index].Status = *cf.EvaluateContract(user, &appraisal, contracts)
		}
	}

	ctx.render(r, w, "user_history.html", struct {
		History    []StatusedAppraisal 	`json:"history"`
		Before     string                 `json:"before"`
		Limit      int                    `json:"limit"`
		HasMore    bool                   `json:"has_more"`
		Next       string                 `json:"next"`
	}{
		history,
		before,
		limit,
		hasMore,
		next,
	})
}
