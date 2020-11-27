package dbmodels

var db IDB

func RegisterDB(idb IDB) {
	db = idb
}

func GetDB() IDB {
	return db
}

type IDB interface {
	ICorporationSigning
	ICorporationManager
	IOrgEmail
	IOrgCLA
	IIndividualSigning
	ICLA
	IVerificationCode
	IPDF
}

type ICorporationSigning interface {
	SignAsCorporation(orgRepo *OrgRepo, info *CorporationSigningInfo) error
	ListCorporationSigning(orgRepo *OrgRepo, language string) ([]CorporationSigningSummary, error)
	GetCorporationSigningSummary(orgRepo *OrgRepo, email string) (CorporationSigningSummary, error)
	GetCorporationSigningDetail(orgRepo *OrgRepo, email string) (CorporationSigningDetail, error)

	UploadCorporationSigningPDF(orgRepo *OrgRepo, adminEmail string, pdf []byte) error
	DownloadCorporationSigningPDF(orgRepo *OrgRepo, email string) ([]byte, error)
}

type ICorporationManager interface {
	AddCorporationManager(orgRepo *OrgRepo, opt []CorporationManagerCreateOption, managerNumber int) error
	DeleteCorporationManager(orgRepo *OrgRepo, emails []string) ([]CorporationManagerCreateOption, error)
	ResetCorporationManagerPassword(orgRepo *OrgRepo, email string, info *CorporationManagerResetPassword) error
	GetCorpManager(*CorporationManagerCheckInfo) ([]CorporationManagerCheckResult, error)
	ListCorporationManager(orgRepo *OrgRepo, email, role string) ([]CorporationManagerListResult, error)
}

type IOrgEmail interface {
	CreateOrgEmail(opt *OrgEmailCreateInfo) error
	GetOrgEmailInfo(email string) (*OrgEmailCreateInfo, error)
}

type IOrgCLA interface {
	GetOrgCLA(string) (OrgCLA, error)
	CreateOrgCLA(OrgCLA) (string, error)
	DeleteOrgCLA(string) error
}

type IIndividualSigning interface {
	SignAsIndividual(orgRepo *OrgRepo, info *IndividualSigningInfo) error
	DeleteIndividualSigning(orgRepo *OrgRepo, email string) error
	UpdateIndividualSigning(orgRepo *OrgRepo, email string, enabled bool) error
	IsIndividualSigned(orgRepo *OrgRepo, email string) (bool, error)
	ListIndividualSigning(orgRepo *OrgRepo, opt *IndividualSigningListOption) ([]IndividualSigningBasicInfo, error)
}

type ICLA interface {
	CreateCLA(CLA) (string, error)
	ListCLA(CLAListOptions) ([]CLA, error)
	GetCLA(string, bool) (CLA, error)
	DeleteCLA(string) error
	ListCLAByIDs(ids []string) ([]CLA, error)
}

type IVerificationCode interface {
	CreateVerificationCode(opt VerificationCode) error
	GetVerificationCode(opt *VerificationCode) error
}

type IPDF interface {
	UploadOrgSignature(orgCLAID string, pdf []byte) error
	DownloadOrgSignature(orgCLAID string) ([]byte, error)
	DownloadOrgSignatureByMd5(orgCLAID, md5sum string) ([]byte, error)

	UploadBlankSignature(language string, pdf []byte) error
	DownloadBlankSignature(language string) ([]byte, error)
}
