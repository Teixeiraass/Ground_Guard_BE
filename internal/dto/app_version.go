package dto

type AppVersionResponse struct {
	// MinimumVersion é a menor versão ainda aceita. Clientes com versão
	// instalada abaixo dessa devem ser bloqueados até atualizar.
	MinimumVersion string `json:"minimum_version"`
	// LatestVersion é a versão mais recente publicada nas lojas. Clientes
	// abaixo dessa (mas acima da mínima) podem continuar usando o app, só
	// recebem um aviso não-bloqueante sugerindo atualizar.
	LatestVersion   string `json:"latest_version"`
	AndroidStoreUrl string `json:"android_store_url"`
	IosStoreUrl     string `json:"ios_store_url"`
}
