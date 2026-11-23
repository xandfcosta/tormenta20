package campaigns

import "errors"

// errNoSuchCampaign é o sinal INTERNO desta cena para "esse id não é de
// campanha nenhuma".
//
// Ele existe porque o `LoadOne` responde 404 e o `LoadList` não, então os dois
// precisam distinguir "não achei" de "deu erro". Para a recusa de ENTRAR o que
// atravessa são os sentinelas do CASO DE USO (`campaign.ErrNoSuchCampaign` e
// irmãos), que esta cena lê direto — o `app/` está abaixo dela (ALE-348).
var errNoSuchCampaign = errors.New("campanha não existe")
