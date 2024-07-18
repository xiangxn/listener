#### 1.编译Trader的GO文件
```
./BuildTraderToGo.sh
```
#### 2.编译项目
如果在Mac下编译可以直接go build，要编译linux版本的执行以下命令：
```
./build-linux.sh
```

#### 3.启动数据库
```
./startdb.sh
```

#### 4.部署到服务器

这步不是必须的，也可以在本地运行
```
./publish.sh
```

#### 5.运行项目
如果未编译可以直接运行“go run main.go”, 已经编译的可以执行下面的命令：
```
./listener [-config config.json]
```

#### 6.一些数据库查询语句
```
db.tokens.aggregate([{$group:{_id:"$address",count:{$sum:1}}},{$match:{count:{$gt:1}}}])

db.pools.aggregate([{$group:{_id:"$address",count:{$sum:1}}},{$match:{count:{$gt:1}}}])

db.prices.aggregate([{$group:{_id:"$pool",count:{$sum:1}}},{$match:{count:{$gt:1}}}])

db.transactions.aggregate([{$match:{buy_pool:"",sell_pool:""}},{$group:{_id:null,maxValue: { $max: "$use_gas" },minValue: { $min: "$use_gas" }}}])

db.transactions.find().sort({created_at:-1}).limit(2)
```

#### 7.一些池地址
```
// base
address public immutable baseToken = 0x4200000000000000000000000000000000000006;
address public immutable borrowPool1 = 0xd0b53D9277642d899DF5C87A3966A349A798F224;
address public immutable borrowPool2 = 0x48413707B70355597404018e7c603B261fcADf3f;

// eth
address public immutable baseToken = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;
address public immutable borrowPool1 = 0x11b815efB8f581194ae79006d24E0d814B7697F6; // ETH/USDT
address public immutable borrowPool2 = 0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640; // USDC/ETH

// USDC/USDT
address public immutable borrowUSD = 0x3416cF6C708Da44DB2624D63ea0AAef7113527C6;

// bsc
address public immutable baseToken = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;
address public immutable borrowPool1 = 0x172fcD41E0913e95784454622d1c3724f546f849;
address public immutable borrowPool2 = 0xf2688Fb5B81049DFB7703aDa5e770543770612C4;
```