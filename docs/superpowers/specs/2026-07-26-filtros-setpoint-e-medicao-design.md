# Filtros de setpoint e de medição — design (2026-07-26)

## Objetivo

Tornar o filtro de setpoint uma opção de configuração de primeira classe
(tipo + tamanho), uniforme nos controladores realimentados e visível na API e na
interface; e mover o filtro de medição de dentro dos controladores para a cadeia
de medição, como opção geral única.

## Escopo A — filtro de setpoint (tipo + tamanho, por controlador)

Suaviza mudanças de referência para reduzir o sobressinal de partida. Presente em
`pid`, `pidff` e `smith` (o Smith hoje recebe degrau cru; passa a ter o mesmo
recurso).

Config (chaves planas, por seção, exemplo na seção `pid`):

```yaml
pid:
  setpoint_filter_type: exponential   # none | moving_average | exponential
  setpoint_filter_size: 2             # em amostras
```

Semântica dos tipos (tamanho sempre em amostras/tiques):

- `none`: passthrough (degrau cru).
- `moving_average`: média móvel de N amostras do sinal de setpoint; um degrau
  vira rampa linear que termina em exatamente N amostras.
- `exponential`: 1ª ordem discreto com a = e^(−1/size) por tique; equivale à
  constante de tempo de `size` amostras. `size: 2` reproduz a convenção validada
  da campanha (τ_f = 2T).

Compatibilidade: `setpoint_filter_tau` (segundos) segue aceito; quando as chaves
novas estão ausentes e `tau > 0`, o comportamento é o exponencial atual
(a = e^(−dt/tau)), idêntico ao validado. Chaves novas têm precedência.
Default sem chave nenhuma: `none`.

Exposição: `GET/PATCH /pid` ganham `setpoint_filter_type` e
`setpoint_filter_size`; persistência do PATCH grava `pid.setpoint_filter_type` e
`pid.setpoint_filter_size`; o painel de configuração da interface mostra os dois
campos para `pid`, `pidff` e `smith`.

## Escopo B — filtro de medição (opção geral da cadeia)

Sai de dentro dos controladores e vira opção única da cadeia de medição,
aplicada à variável de processo antes de qualquer controlador:

```yaml
metrics_poller:
  filter_type: none    # none | moving_average | exponential
  filter_size: 1       # em amostras
```

- Aplicado no caso de uso `processmetrics`, antes de `Compute`; o gauge
  `measured_cpu` passa a registrar o sinal filtrado (com `none`, idêntico a hoje).
- Racional: a identificação lê `measured_cpu`, então o filtro passa a ser
  absorvido pelo modelo identificado como parte da planta medida. Elimina a
  armadilha de dinâmica não modelada dentro da malha do preditor (achado do
  exp. 26/27).
- `pid.filter_size` e `smith.filter_size` são removidos (structs de config,
  `ControllerParams`, DTO e o grupo `SMITH_MODEL` da interface). Chaves antigas
  em configs existentes são ignoradas sem erro.
- Sem endpoint novo para o filtro geral: é decisão de implantação, como a janela
  do `irate` (config apenas).

## Fora de escopo

- Guard do dt gigante no primeiro tique (boot) — dispensado pelo autor.
- Ajuste do texto do PFC (§5.1 menciona "filtro de média móvel usado pelos
  controladores"); sincronizar depois, se o autor quiser.

## Implementação

- `internal/filter`: novo tipo com construtor por (`type`, `size`) cobrindo os
  três comportamentos; média móvel existente reaproveitada.
- `pid.go`/`smith.go`: prefiltro de setpoint via componente novo; remoção do
  filtro de medição interno.
- `processmetrics`: aplica o filtro geral da cadeia.
- `entities`/`dto`/`controllerparams` (usecase): campos novos + persistência;
  remoção de `FilterSize`.
- `web/index.html`: campos `setpoint_filter_*` em pid/pidff/smith; `filter_size`
  sai do grupo do Smith.
- Estilo: sem comentários; funções pequenas com nomes descritivos.

## Testes

- `internal/filter`: passthrough do `none`; degrau→rampa em N amostras da média
  móvel; equivalência do exponencial novo (size) com o comportamento tau atual;
  tamanhos inválidos.
- `pid`/`smith`: referência efetiva rampa após mudança de setpoint em cada tipo;
  Smith com `none` reproduz comportamento atual.
- `processmetrics`: PV filtrada chega ao controlador; gauge reflete o filtrado.
- DTO/handler: GET/PATCH dos campos novos; persistência das chaves novas.
- Regressão: `go test ./...` completo.
