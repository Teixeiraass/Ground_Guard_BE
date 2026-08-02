package routes

type Handlers interface {
	UserHandler
	DeviceHandler
	IrrigationHandler
	HelperHandler
	FaqHandler
	HelpContentHandler
	TutorialHandler
	LegalDocumentHandler
	IrrigationActionHandler
	WebSocketHandler
}
