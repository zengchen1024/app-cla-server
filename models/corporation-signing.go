package models

import (
	"github.com/opensourceways/app-cla-server/dbmodels"
	"github.com/opensourceways/app-cla-server/util"
)

type CorporationSigning = dbmodels.CorpSigningCreateOpt

type CorporationSigningCreateOption struct {
	CorporationSigning

	VerificationCode string `json:"verification_code"`
}

func (this *CorporationSigningCreateOption) Validate(orgCLAID string) IModelError {
	err := checkVerificationCode(this.AdminEmail, this.VerificationCode, orgCLAID)
	if err != nil {
		return err
	}

	if _, err := checkEmailFormat(this.AdminEmail); err != nil {
		return newModelError(ErrNotAnEmail, err)
	}
	return nil

}

func (this *CorporationSigningCreateOption) Create(orgCLAID string) IModelError {
	this.Date = util.Date()

	err := dbmodels.GetDB().SignCorpCLA(orgCLAID, &this.CorporationSigning)
	if err == nil {
		return nil
	}

	if err.IsErrorOf(dbmodels.ErrNoDBRecord) {
		return newModelError(ErrNoLinkOrResigned, err)
	}
	return parseDBError(err)
}

func UploadCorporationSigningPDF(linkID string, email string, pdf *[]byte) error {
	return dbmodels.GetDB().UploadCorporationSigningPDF(linkID, email, pdf)
}

func DownloadCorporationSigningPDF(linkID string, email string) (*[]byte, error) {
	return dbmodels.GetDB().DownloadCorporationSigningPDF(linkID, email)
}

func IsCorpSigningPDFUploaded(linkID string, email string) (bool, error) {
	return dbmodels.GetDB().IsCorpSigningPDFUploaded(linkID, email)
}

func ListCorpsWithPDFUploaded(linkID string) ([]string, error) {
	return dbmodels.GetDB().ListCorpsWithPDFUploaded(linkID)
}

func GetCorporationSigningDetail(platform, org, repo, email string) (string, dbmodels.CorporationSigningSummary, error) {
	return dbmodels.GetDB().GetCorporationSigningDetail(platform, org, repo, email)
}

func GetCorpSigningInfo(platform, org, repo, email string) (string, *dbmodels.CorpSigningCreateOpt, error) {
	return dbmodels.GetDB().GetCorpSigningInfo(platform, org, repo, email)
}

func ListCorpSignings(linkID, language string) ([]dbmodels.CorporationSigningSummary, IModelError) {
	v, err := dbmodels.GetDB().ListCorpSignings(linkID, language)
	if err == nil {
		return v, nil
	}

	if err.IsErrorOf(dbmodels.ErrNoDBRecord) {
		return v, newModelError(ErrNoLink, err)
	}
	return v, parseDBError(err)
}
