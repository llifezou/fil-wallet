# fil-wallet

### FIL hd wallet

#### Already supported:

- wallet:

  - create a mnemonic
  - create wallet
  - export wallet
  - offline signature
  - signature verification
  - balance inquiry
  - transfer amount
  - send transactions
- tool:

  - encode params
  - decode params

#### TODO

- will support multi-signature
- will support fvm calls (in the future)

#### use

- make

  ```
  make all
  ```

  ```
  ./fil-wallet -h  
  NAME:
     fil-wallet - fil wallet

  USAGE:
     fil-wallet [global options] command [command options] [arguments...]

  VERSION:
     v0.0.1

  COMMANDS:
     wallet   fil wallet
     chain    Interact with filecoin blockchain
     help, h  Shows a list of commands or help for one command

  GLOBAL OPTIONS:
     --help, -h     show help (default: false)
     --version, -v  print the version (default: false)

  ```
- Generate mnemonic

  ```
  ./fil-wallet wallet mnemonic
  一定保存好助记词，丢失助记词将导致所有财产损失！
  Be sure to save mnemonic. Losing mnemonic will cause all property damage!

  easily ... ... ... script
  ```
- Generate a wallet

  ```shell
  ./fil-wallet wallet generate --index 1 --type bls
  2022-03-23T20:31:33.924+0800    INFO    wallet  cmd/wallet.go:121       wallet info     {"type": "bls", "index": 1, "path": "m/44'/461'/0'/0/1"}
  f3xxx
  ./fil-wallet wallet generate --index 1 --type secp256k1  
  2022-03-23T20:31:50.479+0800    INFO    wallet  cmd/wallet.go:121       wallet info     {"type": "secp256k1", "index": 1, "path": "m/44'/461'/0'/0/1"}
  f1xxx
  ```
- transfer amount

  ```shell
  ./fil-wallet wallet transfer --from f1xxxx1 --index 1 --gas-premium 11199999 --gas-feecap 11199999 --gas-limit 700000 --nonce 1 --to f1xxxx2 --amount 1

  ```
- balance inquiry

  ```shell
  ./fil-wallet wallet balance f1xxxx
  ```
- encode params

  ```shell
  ./fil-wallet chain encode params --encoding=hex t01000 23 \"t01001\"
  4300e907
  ```
- decode params

  ```shell
  ./fil-wallet chain decode params --encoding=hex t01000 23 4300e907  
  "f01001"
  ```
- offline signature

  ```shell
  ./fil-wallet wallet sign  --index 1 f1em73zadvtid6kvjp22xxb4zbv7srv6uu3whbqvq 4300e907
  2022-04-08T23:35:45.955+0800    INFO    wallet  wallet/account.go:41    wallet info     {"type": "secp256k1", "index": 1, "path": "m/44'/461'/0'/0/1"}
  0159b47df039b230176587f34760466e050c6266c67e97531dde79425e998d95723ada4c816606141304a2b1e3953507597b3b86f8b81262bfba3b61d1a84292d100
  ```
- signature verification

  ```shell
  ./fil-wallet wallet verify --index 1 f1em73zadvtid6kvjp22xxb4zbv7srv6uu3whbqvq 4300e907 0159b47df039b230176587f3476046
  6e050c6266c67e97531dde79425e998d95723ada4c816606141304a2b1e3953507597b3b86f8b81262bfba3b61d1a84292d100
  2022-04-08T23:38:20.917+0800    INFO    wallet  wallet/account.go:41    wallet info     {"type": "secp256k1", "index": 1, "path": "m/44'/461'/0'/0/1"}
  valid signature
  ```
