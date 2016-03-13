package structs

// MapStringString is a map; key=string value=string
type MapStringString map[string]string

// PageInfoStruct contains values that templates may reference
type PageInfoStruct struct {
	Names          MapStringString
	Translated     MapStringString
	DateUTC        string
	Compiled       string
	Locale         string
	LocaleUC       string
	Lang           string
	LangUC         string
	Basename       string
	GitURL         string
	GitRevision    string
	GitLastChanged string
}
