INSERT INTO help_contents (title, slug, category, content, order_number) VALUES
(
  'Entendendo a Saúde do Solo',
  'entendendo-a-saude-do-solo',
  'TIPS',
  $$A Saúde do Solo no seu Perfil mostra quanto tempo, nas últimas 24 horas, cada sensor passou com a umidade dentro da faixa ideal.

• Faixa ideal: começa no limite de umidade que você configurou para a irrigação automática (35% se não configurou) e vai até 80%.
• Ótima: 85% ou mais do tempo na faixa.
• Boa: entre 65% e 85%.
• Atenção: entre 40% e 65%. Vale revisar agendamentos ou o limite de irrigação.
• Crítica: menos de 40%. O solo está passando muito tempo seco ou encharcado.

Sensores offline ou com falha não entram na conta, para não distorcer o resultado.$$,
  1
),
(
  'Onde posicionar o sensor de umidade',
  'onde-posicionar-o-sensor-de-umidade',
  'TIPS',
  $$• Enterre a parte metálica ou sensível do sensor na profundidade das raízes. Para hortaliças, de 5 a 10 cm costuma ser suficiente.
• Coloque o sensor entre as plantas, não encostado no caule nem bem embaixo do gotejador ou aspersor. Assim ele mede a umidade real da área.
• Evite pontos onde a água empoça, a não ser que você queira justamente monitorar esse ponto.
• Mantenha a parte eletrônica do sensor fora da terra e protegida da chuva direta.

Depois de instalar, marque a posição do sensor na aba Mapa. Fica fácil saber qual sensor é qual.$$,
  2
),
(
  'Regar menos vezes, com mais água',
  'regar-menos-vezes-com-mais-agua',
  'TIPS',
  $$Regas curtas e frequentes molham só a superfície, e a raiz fica rasa. Regas mais longas e espaçadas levam a água mais fundo e estimulam raízes profundas, que aguentam melhor o calor.

Para isso no Ground Guard:
• Aumente o Tempo de Irrigação em Preferências de Irrigação.
• Use o Tempo de Espera para o solo secar um pouco entre um ciclo e outro.
• Acompanhe a Saúde do Solo por alguns dias e ajuste aos poucos.$$,
  3
),
(
  'Cobertura do solo segura a umidade',
  'cobertura-do-solo-segura-a-umidade',
  'TIPS',
  $$Cobrir a terra com palha, folhas secas, casca de pinus ou serragem curtida (a chamada cobertura morta) reduz bastante a evaporação. Também protege o solo do sol forte e diminui o mato.

Com a terra coberta, você deve ver a umidade cair mais devagar entre uma rega e outra. É água e energia economizadas.$$,
  4
),
(
  'Cuidado com o excesso de água',
  'cuidado-com-o-excesso-de-agua',
  'TIPS',
  $$Solo sempre encharcado também faz mal: falta oxigênio para as raízes e aparecem fungos e podridão.

Se a Saúde do Solo indicar sensores "encharcados" com frequência:
• Reduza o tempo ou a frequência dos agendamentos.
• Diminua o Limite Diário de Água.
• Verifique se o vaso ou canteiro tem boa drenagem.
• Em dias de chuva, pause os agendamentos daquela zona.$$,
  5
)
ON CONFLICT (slug) DO NOTHING;
