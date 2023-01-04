# fil-wallet

**Warn⚠️: Please save the mnemonic, don't tell anyone, do not upload it to any platform, leaking the mnemonic will cause property damage. This wallet will not save anything, any line of code is under your own control! When using, please ensure that the computer environment is safe!**

**警告⚠️：请保存助记词，不要告诉任何人，不要上传到任何平台，泄露助记词会造成财产损失。这个钱包不会保存任何东西，任何一行代码都在你自己的控制之下！使用时，请保证电脑环境安全！**

**Note⚠️: This open source wallet is free. It also takes no responsibility. Please use it correctly.**

**注意⚠️：这个开源钱包是免费的。它也不承担任何责任。请正确使用。**

### FIL hd wallet

- A hd wallet tool that only needs node url, no need to run a daemon

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
  - multisig transaction
- tool:

  - encode params
  - decode params

#### TODO

- will support fvm calls (in the future)

#### use

```
./fil-wallet wallet -h         
NAME:
   fil-wallet wallet - fil wallet

USAGE:
   fil-wallet wallet command [command options] [arguments...]

COMMANDS:
   mnemonic  Generate a mnemonic
   generate  Generate a key of the given type and index
   sign      Sign a message
   verify    Verify the signature of a message
   balance   Get account balance
   transfer  Transfer funds between accounts
   send      Send funds between accounts
   miner     manipulate the miner actor
   msig      Interact with a multisig wallet
   help, h   Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)

```

```
./fil-wallet wallet miner -h
NAME:
   fil-wallet wallet miner - manipulate the miner actor

USAGE:
   fil-wallet wallet miner command [command options] [arguments...]

COMMANDS:
   new-miner                   new miner, test test test use
   withdraw                    withdraw available balance
   set-owner                   Set owner address (this command should be invoked twice, first with the old owner as the senderAddress, and then with the new owner)
   control                     Manage control addresses
   propose-change-worker       Propose a worker address change
   confirm-change-worker       Confirm a worker address change
   propose-change-beneficiary  Propose a beneficiary address change
   confirm-change-beneficiary  Confirm a beneficiary address change
   help, h                     Shows a list of commands or help for one command

OPTIONS:
   --gas-premium value  specify gas price to use in AttoFIL (default: "0")
   --gas-feecap value   specify gas fee cap to use in AttoFIL (default: "0")
   --gas-limit value    specify gas limit (default: 0)
   --nonce value        specify the nonce to use (default: 0)
   --type value         wallet type, ps: secp256k1, bls (default: "secp256k1")
   --index value        wallet index (default: 0)
   --conf-path value    config.yaml path
   --help, -h           show help (default: false)
```

```
 ./fil-wallet wallet msig -h                                               


NAME:
   fil-wallet wallet msig - Interact with a multisig wallet

USAGE:
   fil-wallet wallet msig command [command options] [arguments...]

COMMANDS:
   create                         Create a new multisig wallet
   inspect                        Inspect a multisig wallet
   propose                        Propose a multisig transaction
   remove-propose                 Propose to remove a signer
   approve                        Approve a multisig message
   cancel                         Cancel a multisig message
   transfer-propose               Propose a multisig transaction
   transfer-approve               Approve a multisig message
   transfer-cancel                Cancel transfer multisig message
   add-propose                    Propose to add a signer
   add-approve                    Approve a message to add a signer
   add-cancel                     Cancel a message to add a signer
   swap-propose                   Propose to swap signers
   swap-approve                   Approve a message to swap signers
   swap-cancel                    Cancel a message to swap signers
   lock-propose                   Propose to lock up some balance
   lock-approve                   Approve a message to lock up some balance
   lock-cancel                    Cancel a message to lock up some balance
   threshold-propose              Propose setting a different signing threshold on the account
   approve-threshold              Approve a message to setting a different signing threshold on the account
   change-owner-propose           Propose an owner address change
   change-owner-approve           Approve an owner address change
   withdraw-propose               Propose to withdraw FIL from the miner
   withdraw-approve               Approve to withdraw FIL from the miner
   change-worker-propose          Propose an worker address change
   change-worker-approve          Approve an owner address change
   confirm-change-worker-propose  Confirm an worker address change
   confirm-change-worker-approve  Confirm an worker address change
   set-control-propose            set control address(-es) propose
   set-control-approve            set control address(-es) approve
   propose-change-beneficiary     Propose a beneficiary address change
   confirm-change-beneficiary     Confirm a beneficiary address change
   help, h                        Shows a list of commands or help for one command

OPTIONS:
   --gas-premium value  specify gas price to use in AttoFIL (default: "0")
   --gas-feecap value   specify gas fee cap to use in AttoFIL (default: "0")
   --gas-limit value    specify gas limit (default: 0)
   --nonce value        specify the nonce to use (default: 0)
   --type value         wallet type, ps: secp256k1, bls (default: "secp256k1")
   --index value        wallet index (default: 0)
   --conf-path value    config.yaml path
   --help, -h           show help (default: false)
```

- build and edit config.yaml

  - `make all`
  - `cp conf/config.yaml.example  conf/config.yaml`
  - run `./fil-wallet -h`
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
- msig

  - msig transfer

  ```
  ./fil-wallet wallet msig --index 0 create --required 3 --from f1xxx0 f1xxx1 f1xxx2 f1xxx3 f1xxx4 f1xxx5
  ./fil-wallet wallet msig --index 1 transfer-propose --from f1xxx1 f2xxx f1xxx 0.05
  ./fil-wallet wallet msig --index 2 transfer-approve --from f1xxx2 f2xxx 1
  ```
