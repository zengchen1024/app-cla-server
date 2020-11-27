package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/opensourceways/app-cla-server/dbmodels"
)

func docFilterOfIndividualSigning(orgRepo *dbmodels.OrgRepo) bson.M {
	return docFilterOfEnabledLink(orgRepo)
}

func elemFilterOfIndividualSigning(email string) bson.M {
	return bson.M{
		fieldCorpID: genCorpID(email),
		"email":     email,
	}
}

func (this *client) SignAsIndividual(orgRepo *dbmodels.OrgRepo, info *dbmodels.IndividualSigningInfo) error {
	signing := dIndividualSigning{
		CLALanguage: "",
		CorpID:      genCorpID(info.Email),
		Name:        info.Name,
		Email:       info.Email,
		Date:        info.Date,
		Enabled:     info.Enabled,
		SigningInfo: info.Info,
	}
	body, err := structToMap(signing)
	if err != nil {
		return err
	}

	docFilter := docFilterOfIndividualSigning(orgRepo)
	arrayFilterByElemMatch(
		fieldSignings, false, elemFilterOfIndividualSigning(info.Email), docFilter,
	)

	f := func(ctx context.Context) error {
		return this.pushArrayElem(ctx, this.individualSigningCollection, fieldSignings, docFilter, body)
	}

	return withContext(f)
}

func (this *client) DeleteIndividualSigning(orgRepo *dbmodels.OrgRepo, email string) error {
	f := func(ctx context.Context) error {
		return this.pullArrayElem(
			ctx, this.individualSigningCollection, fieldSignings,
			docFilterOfIndividualSigning(orgRepo),
			elemFilterOfIndividualSigning(email),
		)
	}

	return withContext(f)
}

func (this *client) UpdateIndividualSigning(orgRepo *dbmodels.OrgRepo, email string, enabled bool) error {
	elemFilter := elemFilterOfIndividualSigning(email)
	docFilter := docFilterOfIndividualSigning(orgRepo)
	arrayFilterByElemMatch(fieldSignings, true, elemFilter, docFilter)

	f := func(ctx context.Context) error {
		return this.updateArrayElem(
			ctx, this.individualSigningCollection, fieldSignings, docFilter,
			elemFilter, bson.M{"enabled": enabled}, false,
		)
	}

	return withContext(f)
}

func (this *client) isIndividualSignedToOrg(orgRepo *dbmodels.OrgRepo, email string) (bool, error) {
	docFilter := docFilterOfIndividualSigning(orgRepo)

	elemFilter := elemFilterOfIndividualSigning(email)
	elemFilter[memberNameOfSignings("enabled")] = true

	signed := false
	f := func(ctx context.Context) error {
		v, err := this.isArrayElemNotExists(
			ctx, this.individualSigningCollection, fieldSignings, docFilter, elemFilter,
		)
		signed = v
		return err
	}

	err := withContext(f)
	return signed, err
}

func (this *client) IsIndividualSigned(orgRepo *dbmodels.OrgRepo, email string) (bool, error) {
	if orgRepo.RepoID == "" {
		return this.isIndividualSignedToOrg(orgRepo, email)
	}

	identity := orgIdentity(orgRepo)

	docFilter := bson.M{
		fieldLinkStatus: linkStatusEnabled,
		fieldOrgIdentity: bson.M{"$in": bson.A{
			genOrgIdentity(orgRepo.Platform, orgRepo.OrgID, ""),
			identity,
		}},
	}

	fieldEnabled := memberNameOfSignings("enabled")

	elemFilter := elemFilterOfIndividualSigning(email)
	elemFilter[fieldEnabled] = true

	var v []cIndividualSigning

	f := func(ctx context.Context) error {
		return this.getArrayElem(
			ctx, this.individualSigningCollection, fieldSignings, docFilter, elemFilter,
			bson.M{
				fieldOrgIdentity: 1,
				fieldEnabled:     1,
			}, &v,
		)
	}

	if err := withContext(f); err != nil {
		return false, err
	}

	num := len(v)
	if num == 0 {
		return false, nil
	}
	if num == 1 {
		return len(v[0].Signings) > 0, nil
	}

	for i := 0; i < len(v); i++ {
		if v[i].OrgIdentity == identity {
			return len(v[i].Signings) > 0, nil
		}
	}

	return false, nil
}

func (this *client) ListIndividualSigning(orgRepo *dbmodels.OrgRepo, opt *dbmodels.IndividualSigningListOption) ([]dbmodels.IndividualSigningBasicInfo, error) {
	docFilter := docFilterOfIndividualSigning(orgRepo)

	arrayFilter := bson.M{}
	if opt.CorporationEmail != "" {
		arrayFilter[fieldCorpID] = genCorpID(opt.CorporationEmail)
	}
	if opt.CLALanguage != "" {
		arrayFilter[fieldCLALanguage] = opt.CLALanguage
	}

	project := bson.M{
		memberNameOfSignings("email"):   1,
		memberNameOfSignings("name"):    1,
		memberNameOfSignings("enabled"): 1,
		memberNameOfSignings("date"):    1,
	}

	var v []cIndividualSigning
	f := func(ctx context.Context) error {
		return this.getArrayElem(
			ctx, this.individualSigningCollection, fieldSignings,
			docFilter, arrayFilter, project, &v)
	}

	if err := withContext(f); err != nil {
		return nil, err
	}

	if len(v) == 0 {
		return nil, nil
	}

	docs := v[0].Signings
	r := make([]dbmodels.IndividualSigningBasicInfo, 0, len(docs))
	for _, item := range docs {
		r = append(r, dbmodels.IndividualSigningBasicInfo{
			Email:   item.Email,
			Name:    item.Name,
			Enabled: item.Enabled,
			Date:    item.Date,
		})
	}

	return r, nil
}
