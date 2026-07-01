package routes

type Handlers interface {
	UserHandler
	DeviceHandler
	IrrigationHandler
	FaqHandler
	HelpContentHandler
	TutorialHandler
	LegalDocumentHandler
	IrrigationActionHandler
}
