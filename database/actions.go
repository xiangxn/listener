package database

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/elliotchance/pie/v2"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// 存储池的表名
	TABLE_POOL = "pools"
	// 存储token的表名
	TABLE_TOKEN = "tokens"
	// 存储价格的表名
	TABLE_PRICE = "prices"
	// 存储交易的表名
	TABLE_TRANSACTION = "transactions"

	FieldTag = "Database"
)

type Actions struct {
	DB     *mongo.Database
	Mctx   context.Context
	Logger logrus.FieldLogger
}

func (a Actions) InitDataBase() {
	colls, err := a.DB.ListCollectionNames(a.Mctx, bson.D{{}})
	if err != nil {
		panic(err)
	}
	// 设置token地址索引
	if !pie.Contains(colls, TABLE_TOKEN) {
		indexName, err := a.DB.Collection(TABLE_TOKEN).Indexes().CreateOne(a.Mctx, mongo.IndexModel{
			Keys:    bson.M{"address": 1},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			panic(err)
		}
		a.Logger.WithFields(logrus.Fields{FieldTag: "initDataBase", "CreateIndex": indexName}).Info()
	}
	// 设置池地址索引
	if !pie.Contains(colls, TABLE_POOL) {
		indexName, err := a.DB.Collection(TABLE_POOL).Indexes().CreateOne(a.Mctx, mongo.IndexModel{
			Keys:    bson.M{"address": 1},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			panic(err)
		}
		a.Logger.WithFields(logrus.Fields{FieldTag: "initDataBase", "CreateIndex": indexName}).Info()
	}
	// 设置池地址索引
	if !pie.Contains(colls, TABLE_PRICE) {
		indexName, err := a.DB.Collection(TABLE_PRICE).Indexes().CreateOne(a.Mctx, mongo.IndexModel{
			Keys:    bson.M{"pool": 1},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			panic(err)
		}
		a.Logger.WithFields(logrus.Fields{FieldTag: "initDataBase", "CreateIndex": indexName}).Info()
	}
}

func (a Actions) GetSimplePools(addrs []string) (pools []dt.SimplePool) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"address": bson.M{"$in": addrs}}
	cursor, err := a.DB.Collection(TABLE_POOL).Find(ctx, filter)
	if err != nil {
		a.Logger.Error("GetSimplePools error: ", err)
		return
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &pools)
	if err != nil {
		a.Logger.Error("GetSimplePools error: ", err)
		return
	}
	return
}

func (a Actions) GetSimplePool(addr string) (pool dt.SimplePool) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()
	err := a.DB.Collection(TABLE_POOL).FindOne(ctx, bson.M{"address": addr}).Decode(&pool)
	if err != nil && err.Error() != "mongo: no documents in result" {
		a.Logger.Error("GetSimplePool error:", err)
		return
	}
	return
}

func (a Actions) SaveTransaction(tx dt.Transaction) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()
	_, err := a.DB.Collection(TABLE_TRANSACTION).InsertOne(ctx, tx)
	if err != nil {
		a.Logger.WithField(FieldTag, "DoSwap").Error(err)
	}
}

func (a Actions) SavePairs(pairs []dt.Pair) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	var wms []mongo.WriteModel
	for _, pair := range pairs {
		if pair.Pool != "" {
			wms = append(wms, mongo.NewUpdateOneModel().
				SetFilter(bson.M{"pool": pair.Pool}).
				SetUpdate(bson.M{"$set": pair, "$inc": bson.M{"updateTimes": 1}}).
				SetUpsert(true))
		}
	}
	if len(wms) < 1 {
		a.Logger.WithField(FieldTag, "SavePairs").Info("没有交易对价格被更新")
		return
	}
	bulkOptions := options.BulkWrite().SetOrdered(false)
	_, err := a.DB.Collection(TABLE_PRICE).BulkWrite(ctx, wms, bulkOptions)
	if err != nil {
		a.Logger.WithField(FieldTag, "SavePairs").Error(err)
	}
}

func (a Actions) SavePair(pool *dt.Pool, price *big.Float, reserve0, reserve1, blockNumber *big.Int, fee float64, dexName string) (pair dt.Pair) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	pair.Pool = pool.Address
	pair.Symbol = fmt.Sprintf("%s/%s", pool.Token0.Symbol, pool.Token1.Symbol)
	pair.Price = tools.ConvertTOFloat64(price)
	pair.Reserve0 = tools.BigIntToFloat64(reserve0, pool.Token0.Decimals)
	pair.Reserve1 = tools.BigIntToFloat64(reserve1, pool.Token1.Decimals)
	pair.BlockNumber = int32(blockNumber.Int64())
	pair.Token0 = pool.Token0.Address
	pair.Token1 = pool.Token1.Address
	pair.Fee = fee
	pair.DexName = dexName

	_, err := a.DB.Collection(TABLE_PRICE).UpdateOne(ctx,
		bson.D{{Key: "pool", Value: pool.Address}},
		bson.D{
			{Key: "$set", Value: pair},
			{Key: "$inc", Value: bson.D{{Key: "updateTimes", Value: 1}}}},
		options.Update().SetUpsert(true))
	if err != nil {
		a.Logger.Error("SavePair error:", err)
	}
	return
}

func (a Actions) GetTokens(addrs []string) (tokens []dt.Token) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"address": bson.M{"$in": addrs}}
	cursor, err := a.DB.Collection(TABLE_TOKEN).Find(ctx, filter)
	if err != nil {
		a.Logger.Error("GetTokens error: ", err)
		return
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &tokens)
	if err != nil {
		a.Logger.Error("GetPools error: ", err)
		return
	}
	return
}

func (a Actions) GetPoolTokens(pool *dt.Pool) bool {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"address": bson.M{"$in": []string{pool.Token0.Address, pool.Token1.Address}}}
	cursor, err := a.DB.Collection(TABLE_TOKEN).Find(ctx, filter)
	if err != nil {
		a.Logger.Error("GetPoolTokens error: ", err)
		return false
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		var token dt.Token
		if err := cursor.Decode(&token); err != nil {
			a.Logger.Error("GetPoolTokens error: ", err)
			return false
		}
		switch token.Address {
		case pool.Token0.Address:
			pool.Token0.Decimals = token.Decimals
			pool.Token0.Name = token.Name
			pool.Token0.Symbol = token.Symbol
			pool.Token0.TotalSupply = token.TotalSupply
			count++
		case pool.Token1.Address:
			pool.Token1.Decimals = token.Decimals
			pool.Token1.Name = token.Name
			pool.Token1.Symbol = token.Symbol
			pool.Token1.TotalSupply = token.TotalSupply
			count++
		default:
			return false
		}
	}
	if count == 2 {
		return true
	} else {
		return false
	}
}

func (a Actions) GetPools(poolAddrs []string) (existingPool []string) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"address": bson.M{"$in": poolAddrs}}
	projection := bson.M{"address": 1, "_id": 0}
	cursor, err := a.DB.Collection(TABLE_POOL).Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		a.Logger.Error("GetPools error: ", err)
		return
	}
	defer cursor.Close(ctx)
	var list []bson.D
	err = cursor.All(ctx, &list)
	if err != nil {
		a.Logger.Error("GetPools error: ", err)
		return
	}
	existingPool = pie.Map(list, func(d bson.D) string { return d[0].Value.(string) })
	return
}

func (a Actions) GetPoolsByTokens(tokens []string) (pools []dt.Pool) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"token0": bson.M{"$in": tokens}, "token1": bson.M{"$in": tokens}}
	cursor, err := a.DB.Collection(TABLE_POOL).Find(ctx, filter)
	if err != nil {
		a.Logger.Error("GetPoolsByTokens error: ", err)
		return
	}
	defer cursor.Close(ctx)

	var list []dt.SimplePool
	err = cursor.All(ctx, &list)
	if err != nil {
		a.Logger.Error("GetPoolsByTokens error: ", err)
		return
	}
	for _, p := range list {
		pool := dt.Pool{
			Factory: p.Factory,
			Address: p.Address,
			Token0:  dt.Token{Address: p.Token0},
			Token1:  dt.Token{Address: p.Token1},
		}
		if a.GetPoolTokens(&pool) {
			pools = append(pools, pool)
		}
	}
	return
}

func (a Actions) SavePools(pools []interface{}) error {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	_, err := a.DB.Collection(TABLE_POOL).InsertMany(ctx, pools)
	return err
}

func (a Actions) SaveTokens(docs []interface{}) error {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()
	_, err := a.DB.Collection(TABLE_TOKEN).InsertMany(ctx, docs)
	return err
}

func (a Actions) GetExistingTokens(tokens []string) (existingToken []string) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"address": bson.M{"$in": tokens}}
	projection := bson.M{"address": 1, "_id": 0}
	cursor, err := a.DB.Collection(TABLE_TOKEN).Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		a.Logger.Error("CheckTokens error: ", err)
		return
	}
	defer cursor.Close(ctx)
	var list []bson.D
	err = cursor.All(ctx, &list)
	if err != nil {
		a.Logger.Error("GetPools error: ", err)
		return
	}
	existingToken = pie.Map(list, func(d bson.D) string { return d[0].Value.(string) })
	return
}

func (a Actions) GetPairsByTokens(tokens []string) (pairs dt.Pairs) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"token0": bson.M{"$in": tokens}, "token1": bson.M{"$in": tokens}}
	cur, err := a.DB.Collection(TABLE_PRICE).Find(ctx, filter)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetPairsByTokens").Error(err)
		return
	}
	err = cur.All(ctx, &pairs)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetPairsByTokens").Error(err)
		return
	}
	return
}

func (a Actions) GetTransactions(ok bool, confirm bool) (txs []dt.Transaction) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"ok": ok, "confirm": confirm, "simulation": false}
	cur, err := a.DB.Collection(TABLE_TRANSACTION).Find(ctx, filter)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetTransactions").Error(err)
		return
	}
	err = cur.All(ctx, &txs)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetTransactions").Error(err)
		return
	}
	return
}

func (a Actions) UpdateTransaction(hash string, confirm bool, gasUsed, gasPrice uint64, income float64, ok bool) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	_, err := a.DB.Collection(TABLE_TRANSACTION).UpdateOne(ctx,
		bson.M{"tx": hash},
		bson.D{
			{Key: "$set", Value: struct {
				Confirm  bool    `bson:"confirm"`
				UseGas   uint64  `bson:"use_gas"`
				GasPrice uint64  `bson:"gas_price"`
				Income   float64 `bson:"income"`
				Ok       bool    `bson:"ok"`
			}{
				Confirm:  confirm,
				UseGas:   gasUsed,
				GasPrice: gasPrice,
				Income:   income,
				Ok:       ok,
			}},
		})
	if err != nil {
		a.Logger.Error("UpdateTransaction error:", err)
	}
}

func (a Actions) GetToken(addr string) (token dt.Token) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	err := a.DB.Collection(TABLE_TOKEN).FindOne(ctx, bson.M{"address": addr}).Decode(&token)
	if err != nil && err.Error() != "mongo: no documents in result" {
		a.Logger.Error("GetToken error:", err)
	}
	return
}

func (a Actions) GetGas(buyPool, sellPool string) (min, max int64) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	var results []bson.M
	pipeline := []bson.M{
		{"$match": bson.M{"buy_pool": buyPool, "sell_pool": sellPool}},
		{"$group": bson.M{"_id": nil, "maxGas": bson.M{"$max": "$use_gas"}, "minGas": bson.M{"$min": "$use_gas"}}},
	}
	cursor, err := a.DB.Collection(TABLE_TRANSACTION).Aggregate(ctx, pipeline)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetGas").Error(err)
		return
	}
	if err = cursor.All(ctx, &results); err != nil {
		a.Logger.WithField(FieldTag, "GetGas").Error(err)
		return
	}
	if len(results) > 0 && results[0]["minGas"] != nil && results[0]["maxGas"] != nil {
		min = results[0]["minGas"].(int64)
		max = results[0]["maxGas"].(int64)
	}
	return
}

func (a Actions) GetFailTransacttionCount(buyPool, sellPool string) int {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	var txs []dt.Transaction
	pools := []string{buyPool, sellPool}
	filter := bson.M{"buy_pool": bson.M{"$in": pools}, "sell_pool": bson.M{"$in": pools}, "ok": false, "confirm": true}
	cur, err := a.DB.Collection(TABLE_TRANSACTION).Find(ctx, filter)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetFailTransacttionCount").Error(err)
		return 0
	}
	err = cur.All(ctx, &txs)
	if err != nil {
		a.Logger.WithField(FieldTag, "GetFailTransacttionCount").Error(err)
		return 0
	}
	return len(txs)
}

func (a Actions) SearchTransacttion(simulation bool, start time.Time, end time.Time) (txs []dt.Transaction) {
	ctx, cancel := context.WithCancel(a.Mctx)
	defer cancel()

	filter := bson.M{"simulation": simulation, "created_at": bson.M{"$gte": start, "$lt": end}}
	cur, err := a.DB.Collection(TABLE_TRANSACTION).Find(ctx, filter)
	if err != nil {
		a.Logger.WithField(FieldTag, "SearchTransacttion").Error(err)
		return
	}
	err = cur.All(ctx, &txs)
	if err != nil {
		a.Logger.WithField(FieldTag, "SearchTransacttion").Error(err)
		return
	}
	return
}
