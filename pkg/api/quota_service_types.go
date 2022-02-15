package api

const (
	AMSQuotaType                 QuotaType = "ams"
	QuotaManagementListQuotaType QuotaType = "quota-management-list"
	UndefinedQuotaType           QuotaType = ""
)

type QuotaType string

func (quotaType QuotaType) String() string {
	return string(quotaType)
}
