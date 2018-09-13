package usertempmatch

import (
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"

	"github.com/eure/si2018-second-half-2/entities"
	tokenlib "github.com/eure/si2018-second-half-2/libs/token"
	// userlib "github.com/eure/si2018-second-half-2/libs/user"
	"github.com/eure/si2018-second-half-2/repositories"
	si "github.com/eure/si2018-second-half-2/restapi/summerintern"
)

func GetTempMatch(p si.GetTempMatchParams) middleware.Responder {
	s := repositories.NewSession()

	// Validation
	t := p.Token
	if res := ValidateGetTempMatch(s, t); res != nil {
		return res
	}

	//

	return si.NewGetTempMatchOK()
}

func PostTempMatch(p si.PostTempMatchParams) middleware.Responder {
	s := repositories.NewSession()

	// Validation
	t := p.Token
	if res := ValidatePostTempMatch(s, t); res != nil {
		return res
	}

	// Get me
	me, err := tokenlib.GetUserByToken(s, t)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error :: Meの取得に失敗しました",
			})
	}
	if me == nil {
		return si.NewPostTempMatchUnauthorized().WithPayload(
			&si.PostTempMatchUnauthorizedBody{
				Code:    "401",
				Message: "Unauthorized :: Tokenが無効です",
			})
	}

	// Check whether I matched (Male)
	if me.Gender == "M" {
		matchRepo := repositories.NewUserMatchRepository(s)
		matchedIDs, err := matchRepo.FindAllByUserID(me.ID)
		if err != nil {
			return si.NewPostTempMatchInternalServerError().WithPayload(
				&si.PostTempMatchInternalServerErrorBody{
					Code:    "500",
					Message: "Internal Server Error",
				})
		}
		if matchedIDs != nil {
			return si.NewPostTempMatchBadRequest().WithPayload(
				&si.PostTempMatchBadRequestBody{
					Code:    "400",
					Message: "Bad Request :: You (Male) already matched to someone",
				})
		}
	}

	// きょうすでに使ったかどうか確認
	waitRepo := repositories.NewUserWaitTempMatchRepository(s)
	isMatched, err := waitRepo.IsMatchedToday(me.ID)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}
	if isMatched == true {
		return si.NewPostTempMatchBadRequest().WithPayload(
			&si.PostTempMatchBadRequestBody{
				Code:    "400",
				Message: "Bad Request :: You already temp matched today",
			})
	}

	// Check if you are active
	activeEnt, err := waitRepo.GetActive(*me)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}
	if activeEnt == nil {
		// Create UserWaitTempMatch entities for me
		now := strfmt.DateTime(time.Now())
		waitEnt := entities.UserWaitTempMatch{
			UserID:     me.ID,
			Gender:     me.Gender,
			IsMatched:  false,
			IsCanceled: false,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		err = waitRepo.Create(waitEnt)
		if err != nil {
			return si.NewPostTempMatchInternalServerError().WithPayload(
				&si.PostTempMatchInternalServerErrorBody{
					Code:    "500",
					Message: "Internal Server Error :: Failed to wait temp match",
				})
		}
	}

	// Search suited user for me
	partnerID, err := waitRepo.SearchPartner(*me)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error :: Failed to search partner",
			})
	}
	if partnerID == 0 {
		var emptyEnt entities.UserTempMatch
		sEnt := emptyEnt.Build()
		return si.NewPostTempMatchOK().WithPayload(&sEnt)
	}

	// Create temp match
	now := strfmt.DateTime(time.Now())
	tempmatchEnt := entities.UserTempMatch{
		UserID:    me.ID,
		PartnerID: partnerID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tempmatchRepo := repositories.NewUserTempMatchRepository(s)
	err = tempmatchRepo.Create(tempmatchEnt)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	// TODO: Create したものを（TempMatch）をとってくる作業が必要
	updatedWaitEnt, err := tempmatchRepo.GetLatest(tempmatchEnt.UserID, tempmatchEnt.CreatedAt)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}
	if updatedWaitEnt == nil {
		return si.NewPostTempMatchBadRequest().WithPayload(
			&si.PostTempMatchBadRequestBody{
				Code:    "400",
				Message: "Bad Request :: Failed to get updated temp match",
			})
	}

	// Update UserWaitTempMatch.IsMatch -> true
	activeEnt.IsMatched = true
	err = waitRepo.Update(activeEnt)
	if err != nil {
		return si.NewPostTempMatchInternalServerError().WithPayload(
			&si.PostTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	sEnt := updatedWaitEnt.Build()
	return si.NewPostTempMatchOK().WithPayload(&sEnt)
}

func PutTempMatch(p si.PutTempMatchParams) middleware.Responder {
	s := repositories.NewSession()

	// Validation
	t := p.Token
	if res := ValidatePutTempMatch(s, t); res != nil {
		return res
	}

	// Get me
	me, err := tokenlib.GetUserByToken(s, t)
	if err != nil {
		return si.NewPutTempMatchInternalServerError().WithPayload(
			&si.PutTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error :: Meの取得に失敗しました",
			})
	}
	if me == nil {
		return si.NewPutTempMatchUnauthorized().WithPayload(
			&si.PutTempMatchUnauthorizedBody{
				Code:    "401",
				Message: "Unauthorized :: Tokenが無効です",
			})
	}

	// Get latest My UserWaitTempMatch
	r := repositories.NewUserWaitTempMatchRepository(s)
	latestUser, err := r.GetLatestByUserID(me.ID)
	if err != nil {
		return si.NewPutTempMatchInternalServerError().WithPayload(
			&si.PutTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}
	if latestUser == nil {
		return si.NewPutTempMatchBadRequest().WithPayload(
			&si.PutTempMatchBadRequestBody{
				Code:    "400",
				Message: "Bad Request",
			})
	}

	// Cancel to wait
	latestUser.IsCanceled = true
	err = r.Update(latestUser)
	if err != nil {
		return si.NewPutTempMatchInternalServerError().WithPayload(
			&si.PutTempMatchInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	return si.NewPutTempMatchOK().WithPayload(
		&si.PutTempMatchOKBody{
			Code:    "200",
			Message: "Canceled",
		})
}
