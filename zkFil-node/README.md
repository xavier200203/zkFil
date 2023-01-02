
# zkFil: A decentralized system for data exchange


**Available in [ [English](README.md) | [中文](README.zh.md) ]**

[![All Contributors](https://img.shields.io/badge/all_contributors-10-orange.svg?style=flat-square)](#contributors)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Discord](https://img.shields.io/discord/586796918248570890.svg)](https://discord.gg/tfUH886)
[![Build Status](https://travis-ci.org/xuxinlai2002/zkFil/zkFil-node.svg?branch=master)](https://travis-ci.org/xuxinlai2002/zkFil/zkFil-node)


## Overview

zkFil is a decentralized platform for data exchange between *untrusted parties* realizing "Payment on Delivery" without any *trusted third party*. Instead, zkFil uses blockchain (e.g., Ethereum) as a *trustless third party* to ensure fairness that no party can cheat during data exchange. Moreover, zkFil is concerned with users' privacy, hiding the intention of users to either blockchain miners or other parties. Any seller can publish data for:

- ***Data Downloading***: Buyers may pay-and-download a data file from a data seller. zkFil supports data fragments downloading, i.e., buyers may download specific data chunks in one batched transaction. 

- ***Data Query***:  zkFil supports structured data; e.g., the seller organizes data as tables. Multiple columns can be selected as indexed-columns, such that users may pay-and-query records in the table with one or more keywords, and get the records matched. zkFil ensures that the query results are trustworthy, i.e. (i) if data seller replies with n records, it is impossible that more records are matching that keyword in the table; (ii) these n records are precisely in the table, and any forged records cannot be allowed. 

The three main issues being tackled by zkFil are

+ The data is precisely what the buyer wants before payment,
+ The data must be delivered when the buyer pays,
+ The data won't be leaked before being paid.

A cryptographic protocol, PoD (proof of delivery), is developed to try to solve the issues, ensuring **fairness** between data buyers and sellers. The protocol is zero-knowledge and provable secure (*ongoing work*). See our [technical paper](https://xuxinlai2002/zkFil.github.io/zkFil-node/paper.pdf) for more information. 

zkFil is practical and efficient. It could deliver data with TBs in theory. See [performance evaluation section](#Performance) below.

[![asciicast-gif](img/demo.min.gif)](https://asciinema.org/a/251240?autoplay=1&speed=2.71828182846)

## Highlights 

+ Decentralization:  zkFil uses smart contracts on Ethereum as the trustless third party. In theory, zkFil can be deployed on any blockchains with basic smart contract support. The gas cost in transactions of data exchange is moderate, and the size of data can be up to TBs.
+ Atomic-swap:  zkFil supports atomic-swap (as in [ZKCP](https://en.bitcoin.it/wiki/Zero_Knowledge_Contingent_Payment)).
+ Large data file support.  zkFil supports delivering large data file within one transaction. See [performance evaluation section](#Performance).
+ Data query by keywords:  zkFil supports pay-and-query. Before locating the records interested, a buyer may query for one or more keywords.
+ Privacy protection: The request of a buyer may be sensitive under some circumstances, the buyer can obfuscate her real intention by adding a few unrelated requests. Then the seller has to respond to all requests without knowing which one is real from the buyer, but she does know that only one response can be visible to the buyer since the buyer only paid for one request. 
+ Inspection of goods:  zkFil supports the inspection of goods for a buyer at any scale natively. The buyer can randomly select any piece of data at any location and takes it as a sample to check whether it is something she wants or not. Then, the buyer can continue to buy a large amount of data after a satisfied inspection. zkFil does not set a limit for the number of times a buyer could request for inspection. zkFil also ensures that every piece of data in every inspection coming from the same data set, including the final batch purchase.

## Project Structure

<p align="center"> <img src="img/overview.svg"> </p>

- [zkFil-node](https://github.com/xuxinlai2002/zkFil/zkFil-node) Node application written in Golang for sellers (Alice) and buyers (Bob). It deals with communication, smart contract calling, data transferring, and other zkFil protocol interactions.
- [zkFil-lib](https://github.com/xuxinlai2002/zkFil/zkFil-lib) zkFil core library written in C++ shipping with Golang bindings.
- [zkFil-contract](https://github.com/xuxinlai2002/zkFil/zkFil-contract) Smart contracts for zkFil *Decentralized Exchange*.

## Workflow and how it works

We briefly describe the workflow of transactions on zkFil by a simplified version of the PoD protocol. 

TODO: re-draw this diagram.

![](img/regular.png)

#### Data initialization

Data must be processed before being sold. Alice needs to compute the authenticators of data and the Merkle root of them. Authenticators are for data contents and origin verification (even if the data were encrypted). zkFil supports two modes: plain mode and table mode. 

+ plain mode
+ table mode (CSV files)

For tabulated data, each row is a record with fixed columns. The buyer may send queries with keywords. Note that the columns must be specified before data initialization to supports keywords.

#### Data transaction

We present three variant protocols, PoD-AS, PoD-AS* and PoD-CR, used for different purposes. See the [performance evaluation section](#Performance) and our [technical paper](https://xuxinlai2002/zkFil.github.io/zkFil-node/paper.pdf) for detailed specification and comparison.

For simplicity, we introduce two main types of trading mode for data delivery.

+ Atomic-swap mode

1. Bob sends request w.r.t. a data tag
2. Alice sends encrypted data to Bob (by a one-time random key)
3. Bob verifies the *encrypted* data with tag by using ZKP.
4. Bob accepts the data and submits a receipt to the contract (blockchain).
5. Alice checks the receipt and then reveals the key (for encrypting the data)
6. Contract (blockchain) verifies if the key matches the receipt and output "accept"/"reject."

+ Complaint mode (inspired by Fairswap)

1. Bob sends request w.r.t. a data tag
2. Alice sends encrypted data to Bob (by a one-time random key)
3. Bob verifies the *encrypted* data with tag by using ZKP.
4. Bob accepts the data and submits a receipt to the contract(blockchain).
5. Alice checks the receipt and then reveals the key (for encrypting the data)
6. Bob decrypts the data by the key and submits proof of misbehavior to the contract(blockchain) if he finds that Alice was cheating.

### Theories behind

For fairness and security, the protocol ensures the following requirements:

- {1} Contract (blockchain) cannot learn anything about the data, or encrypted data
- {2} Bob must submit a correct receipt to get the real key
- {3} Bob must pay before obtaining the key
- {4} Bob cannot learn anything from the encrypted data
- {5} Alice cannot reveal a fake key, which would be ruled out by the verification algorithm of contract(blockchain)
- {6} Alice cannot send junk data to Bob, who cannot cheat when verifying data tag.

To ensure **{1, 4, 6}**, we use ZKP based on Pedersen commitments (which is additively homomorphic) with one-time-pad encryption, allowing buyers to verify the data without the help of others. A smart contract is used to exchange crypto coins with keys to ensure **{2, 3, 5}** in the way of transparent, predictable and formally verified (*ongoing work*).

We use *verifiable random function*, VRF, to support queries with keywords. Currently, zkFil only supports exact keyword matching. zkFil adopts *oblivious transfer*, OT, to support privacy-preserving queries.

## Play With It

### Build

*WIP: A building script for all of these steps*

#### 1. Build zkFil-lib

Dependencies of zkFil-lib could be found [here](https://github.com/xuxinlai2002/zkFil/zkFil-lib#dependencies). Make sure you install them first.

```shell
# Download zkFil-lib code
mkdir zkFil && cd zkFil
git clone https://github.com/xuxinlai2002/zkFil/zkFil-lib.git

# Pull libsnark submodule
cd zkFil-lib
git submodule init && git submodule update
cd depends/libsnark
git submodule init && git submodule update

# Build libsnark
mkdir build && cd build
# - On Ubuntu
cmake -DCMAKE_INSTALL_PREFIX=../../install -DMULTICORE=ON -DWITH_PROCPS=OFF -DWITH_SUPERCOP=OFF -DCURVE=MCL_BN128 ..
# - Or for macOS (see https://github.com/scipr-lab/libsnark/issues/99#issuecomment-367677834)
CPPFLAGS=-I/usr/local/opt/openssl/include LDFLAGS=-L/usr/local/opt/openssl/lib PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig cmake -DCMAKE_INSTALL_PREFIX=../../install -DMULTICORE=OFF -DWITH_PROCPS=OFF -DWITH_SUPERCOP=OFF -DCURVE=MCL_BN128 ..
make && make install

# Build zkFil-lib
cd ../../..
make

# These files should be generated after successful build.
# pod_setup/pod_setup
# pod_publish/pod_publish
# pod_core/libpod_core.so
# pod_core/pod_core

cd pod_go
export GO111MODULE=on
make test
```

#### 2. Build zkFil-node

```shell
cd zkFil
git clone https://github.com/xuxinlai2002/zkFil/zkFil-node.git
cd zkFil-node
export GO111MODULE=on
make
```

### Have Fun

#### 1. Setup

We need a [trusted setup](https://z.cash/technology/paramgen/) to generate zkFil zkSNARK parameters.

For convenience and testing purposes, we could download it from [zkFil-params](https://github.com/xuxinlai2002/zkFil/zkFil-params) repo.

```shell
cd zkFil-node
mkdir -p zkFilParam/zksnark_key
cd zkFilParam/zksnark_key
# Download zkSNARK pub params, see https://github.com/xuxinlai2002/zkFil/zkFil-params
wget https://raw.githubusercontent.com/xuxinlai2002/zkFil/zkFil-params/master/zksnark_key/atomic_swap_vc.pk
wget https://raw.githubusercontent.com/xuxinlai2002/zkFil/zkFil-params/master/zksnark_key/atomic_swap_vc.vk
```

#### 2. Run node

```shell
cd zkFil-node
make run
# A config file named basic.json is generated on local
```
> Examples: [`basic.json`](examples/basic.json) - Some basic configs of zkFil-node program.

Tips: 

You should specify `LD_LIBRARY_PATH` for `libpod_core` when executing `zkFil-node` on Linux. On macOS, you should use `DYLD_LIBRARY_PATH` instead. Check `Makefile` for examples. For convenience, you could set `LD_LIBRARY_PATH` as an environment variable.

```shell
# On Linux
export LD_LIBRARY_PATH=<YOUR_PATH_TO_LIBPOD_CORE>

# Or on macOS
export DYLD_LIBRARY_PATH=<YOUR_PATH_TO_LIBPOD_CORE>
```

#### 3. Save keystore & get some ETH

- https://faucet.ropsten.be/
- https://faucet.metamask.io/

Tips: A new Ethereum account is generated after the first boot of zkFil-node. You could read it from the terminal screen or keystore file. Keep your keystore safe. You must have some ETH balance in your Ethereum address for smart contract interaction. Get some for the test from a ropsten Ethereum faucet.

#### 4. As a seller: init data & publish 

Open a new terminal

```shell
# On Linux
export LD_LIBRARY_PATH=`pwd`/../zkFil-lib/pod_core/

# Or on macOS
export DYLD_LIBRARY_PATH=`pwd`/../zkFil-lib/pod_core/

cd zkFil-node
mkdir bin
cp ../zkFil-lib/pod_publish/pod_publish ./bin

wget -O test.txt https://www.gutenberg.org/files/11/11-0.txt

# cp examples/init.json .
./zkFil-node -o initdata -init init.json
# You should get the sigma_mkl_root from logs
# export sigma_mkl_root=<YOUR_SIGMA_MKL_ROOT>
./zkFil-node -o publish -mkl $sigma_mkl_root -eth 200
# You should get the publish transaction hash
```
> Examples: [init.json](examples/init.json) - Use this to describe your data for sell.

Tips: For a test, you could use the same Ethereum account for selling and buying. You could also host a zkFil-node and publish your data description to the [community](https://discord.gg/tfUH886) for trade testing.

Here is everything that you need to let others know.

```
- Your IP address
- Your ETH address
- Data sigma_mkl_root for trade
- Data description
- Data bulletin file
- Data public info 
```

You could get `bulletin` and `public info` of your data for publishing in path `zkFil-node/A/publish/$sigma_mkl_root/`.

```
├── bulletin
├── extra.json
├── private
│   ├── matrix
│   └── original
├── public
│   ├── sigma
│   └── sigma_mkl_tree
└── test.txt
```

#### 5. As a buyer: deposit to contract

You want to buy some data you interested in from a seller. You could deposit some ETH to *zkFil exchange contract* first. Your money is still yours before you get the data you want.

```shell
./zkFil-node -o deposit -eth 20000 -addr $SELLER_ETH_ADDR
# You should get the deposit transaction hash
```

#### 6. As a buyer: purchase data

You'll make a purchase request to a seller. For convenience, you could fill in some basic info of the seller in the config file.

```shell
# For test, you could simply copy public info of data from seller folder to project root path.
# cp A/publish/$sigma_mkl_root/bulletin .
# cp -r A/publish/$sigma_mkl_root/public .
./zkFil-node -o purchase -c config.json
# You should get the decrypted data in B/transaction/<session_id> folder
```
> Examples: [config.json](examples/config.json) - Use this to describe data you are going to buy.

Tips:
1. PoD-AS protocol is much more suitable for permissioned blockchain and only supports up to about 350 KiB on the Ethereum network for the moment due to the block gas limit.
2. For PoD-AS* protocol, there is no bottleneck in on-chain computation and smart contracts could verify data of unlimited size. But it will have slower off-chain computation.
3. If the PoD-CR protocol is selected, zkFil-node complains to the contract automatically with proof proving that the seller is dishonest. As a result, a dishonest seller would never profit from misbehavior.

TODO: Add more examples about a query or private query of table data, and other operations.

## Performance

#### Test Environment

- OS: Ubuntu 16.04.6 LTS x86_64
- CPU Model: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
- CPU Thread Count: 12
- Memory: 32605840 kB

#### Basic Info

|  Protocol  | Throughput |   Communication   |   Gas Cost (Ethereum)   | Data/Tx (Ethereum) |
| :----: | :----------------: | :---------------------: | :---------------------: | :---------------------: |
| PoD-CR |        3.39 MiB/s       |        $O(2n)$        | $O(\log{}n)$ |         < 100 TiB         |
| PoD-AS |        3.91 MiB/s       |    $O(2n)$    |    $O(n)$    |        < 350 KiB        |
| PoD-AS* |    35 KiB/s    |    $O(2n)$    |    $O(1)$    |        Unlimited        |

PoD-AS supports the fastest data delivery with O(n) on-chain computation. The variant is suitable for permissioned blockchain, where the performance (TPS) is high and the computation cost of the smart contract is pretty low.

PoD-AS* is using zkSNARKs to reduce on-chain computation to O(1), but with slower off-chain delivery.

PoD-CR supports fast data delivery and small on-chain computation O(log(n)).

#### Benchmark Results

- Data size: 1024 MiB
- File type: plain
- s: 64
- omp_thread_num: 12

|      Protocol      | Prover (s) | Verifier (s) | Decrypt (s) | Communication Traffic (MiB) | Gas Cost |
| :------------: | :--------: | :----------: | :---------: | :-------------------------: | :------: |
| PoD-CR |    124     |     119      |     82      |            2215             | 159,072  |
|  PoD-AS   |    130     |     131      |    4.187    |            2215             |   `*`    |
|  PoD-AS*   |    34540     |     344      |    498    |            2226             |   183,485   |


`*` PoD-AS protocol does not support 1 GiB file on the Ethereum network at present.

#### Gas Cost on Ethereum

PoD-CR Protocol            |  PoD-AS Protocol      |  PoD-AS* Protocol
:-------------------------:|:-------------------------:|:-------------------------:
![](img/Gas-Cost-vs-Data-Size-Batch1.svg)  | ![](img/Gas-Cost-vs-Data-Size-Batch2.svg) | ![](img/Gas-Cost-vs-Data-Size-Batch3.svg) 

## Learn more?

+ White paper: an overview introduction of the zkFil system.
+ [Technical paper](https://xuxinlai2002/zkFil.github.io/zkFil-node/paper.pdf): a document with theoretic details to those who are interested in the theory we are developing.
+ Community: join us on [*Discord*](https://discord.gg/tfUH886) and follow us on [*Twitter*](https://twitter.com/SECBIT_IO) please!

## Related projects

+ Fairswap:  https://github.com/lEthDev/FairSwap
+ ZKCP: https://en.bitcoin.it/wiki/Zero_Knowledge_Contingent_Payment
+ Paypub: https://github.com/unsystem/paypub

## License

[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore -->
<table>
  <tr>
    <td align="center"><a href="https://github.com/huyuguang"><img src="https://avatars1.githubusercontent.com/u/2227368?v=4" width="100px;" alt="Hu Yuguang"/><br /><sub><b>Hu Yuguang</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=huyuguang" title="Code">💻</a> <a href="#ideas-huyuguang" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=huyuguang" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/x0y1"><img src="https://avatars1.githubusercontent.com/u/33647147?v=4" width="100px;" alt="polymorphism"/><br /><sub><b>polymorphism</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=x0y1" title="Code">💻</a> <a href="#ideas-x0y1" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=x0y1" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/10to4"><img src="https://avatars2.githubusercontent.com/u/35983442?v=4" width="100px;" alt="even"/><br /><sub><b>even</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=10to4" title="Code">💻</a> <a href="#ideas-10to4" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=10to4" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/zer0to0ne"><img src="https://avatars3.githubusercontent.com/u/36526113?v=4" width="100px;" alt="zer0to0ne"/><br /><sub><b>zer0to0ne</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=zer0to0ne" title="Code">💻</a> <a href="#ideas-zer0to0ne" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=zer0to0ne" title="Documentation">📖</a></td>
    <td align="center"><a href="https://twitter.com/ErrNil"><img src="https://avatars0.githubusercontent.com/u/36690236?v=4" width="100px;" alt="p0n1"/><br /><sub><b>p0n1</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=p0n1" title="Code">💻</a> <a href="#ideas-p0n1" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=p0n1" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/aphasiayc"><img src="https://avatars3.githubusercontent.com/u/24490151?v=4" width="100px;" alt="aphasiayc"/><br /><sub><b>aphasiayc</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=aphasiayc" title="Code">💻</a> <a href="#ideas-aphasiayc" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=aphasiayc" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/Vawheter"><img src="https://avatars1.githubusercontent.com/u/24186846?v=4" width="100px;" alt="Vawheter"/><br /><sub><b>Vawheter</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=Vawheter" title="Code">💻</a> <a href="#ideas-Vawheter" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=Vawheter" title="Documentation">📖</a></td>
  </tr>
  <tr>
    <td align="center"><a href="https://github.com/hzzhang"><img src="https://avatars3.githubusercontent.com/u/1537758?v=4" width="100px;" alt="Haozhong Zhang"/><br /><sub><b>Haozhong Zhang</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=hzzhang" title="Code">💻</a> <a href="#ideas-hzzhang" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=hzzhang" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/pkuzhangchao"><img src="https://avatars2.githubusercontent.com/u/2003972?v=4" width="100px;" alt="Chao Zhang"/><br /><sub><b>Chao Zhang</b></sub></a><br /><a href="#ideas-pkuzhangchao" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=pkuzhangchao" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/gy001"><img src="https://avatars0.githubusercontent.com/u/9260429?v=4" width="100px;" alt="Yu Guo"/><br /><sub><b>Yu Guo</b></sub></a><br /><a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=gy001" title="Code">💻</a> <a href="#ideas-gy001" title="Ideas, Planning, & Feedback">🤔</a> <a href="https://github.com/xuxinlai2002/zkFil/zkFil-node/commits?author=gy001" title="Documentation">📖</a></td>
  </tr>
</table>

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!
