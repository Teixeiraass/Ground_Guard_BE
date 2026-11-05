package routes

type Handlers interface {
	UserHandler
	DeviceHandler
	ControllerDeviceHandler
	IrrigationHandler
	HelperHandler
	FaqHandler
	HelpContentHandler
	TutorialHandler
	LegalDocumentHandler
	IrrigationActionHandler
	IrrigationScheduleHandler
	StatisticsHandler
	WebSocketHandler
	NotificationHandler
	DeviceGroupHandler
	AdminHandler
	PasswordResetHandler
	SupportHandler
	FirmwareHandler
	WeatherHandler
}
