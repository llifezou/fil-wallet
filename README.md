# fil-wallet
### Fil hd wallet

#### Already supported:

- create a mnemonic
- create wallet
- export wallet

#### TODO

- will support offline signature
- will support signature verification
- will support balance inquiry
- will support transfer
- will support  send transactions
- will support multi-signature
- will support fvm calls (in the future)

#### use

- make

  ```
  go mod tidy -go=1.16 && go mod tidy -go=1.17
  go build -o fil-wallet main.go
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

-  Generate a wallet

  ```
  ./fil-wallet wallet generate --index 1 --type bls
  2022-03-23T20:31:33.924+0800    INFO    wallet  cmd/wallet.go:121       wallet info     {"type": "bls", "index": 1, "path": "m/44'/461'/0'/0/1"}
  f3wgai4sgumcucz5vxxosxfgsdm43x3jg2o6xywqcmdjlvxarfua7n5y5bvtvf4mnboq5jvt6lva3pkp5htj7a
  
  ./fil-wallet wallet generate --index 1 --type secp256k1  
  2022-03-23T20:31:50.479+0800    INFO    wallet  cmd/wallet.go:121       wallet info     {"type": "secp256k1", "index": 1, "path": "m/44'/461'/0'/0/1"}
  f1em73zadvtid6kvjp22xxb4zbv7srv6uu3whbqvq
  ```

  
