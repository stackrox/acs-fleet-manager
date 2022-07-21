package api

// AMSQuotaType ...
const (
	AMSQuotaType                 QuotaType = "ams"
	QuotaManagementListQuotaType QuotaType = "quota-management-list"
	UndefinedQuotaType           QuotaType = ""
)

// QuotaType ...
type QuotaType string

// String ...
func (quotaType QuotaType) String() string {
	return string(quotaType)
}
