package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/database"
	dt "github.com/xiangxn/listener/types"
	"go.mongodb.org/mongo-driver/bson"
)

func TestClass(t *testing.T) {
	cfg := config.GetConfig("../config.yaml")
	db := database.GetClient(cfg).Database(fmt.Sprintf("%slistener", cfg.NetName))
	ctx := context.Background()
	var data dt.Pairs
	tokens := []string{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xed4e879087ebD0e8A77d66870012B5e0dffd0Fa4"}
	filter := bson.M{"token0": bson.M{"$in": tokens}, "token1": bson.M{"$in": tokens}}
	cur, err := db.Collection(database.TABLE_PRICE).Find(ctx, filter)
	if err != nil {
		return
	}
	err = cur.All(ctx, &data)
	if err != nil {
		return
	}
	fmt.Println(data[0].Price, data[1].Price)
	data = ChangeTheData(data)
	fmt.Println(data[0].Price, data[1].Price)
}

func ChangeTheData(data dt.Pairs) dt.Pairs {
	d := data[0]
	d.Price = 123.55
	return data
}
