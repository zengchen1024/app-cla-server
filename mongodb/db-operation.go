package mongodb

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/opensourceways/app-cla-server/dbmodels"
	"github.com/opensourceways/app-cla-server/util"
)

func condOfArrayElemFilter(filterOfArray bson.M) bson.M {
	cond := make(bson.A, 0, len(filterOfArray))
	for k, v := range filterOfArray {
		cond = append(cond, bson.M{"$eq": bson.A{"$$this." + k, v}})
	}

	if len(filterOfArray) == 1 {
		return cond[0].(bson.M)
	}

	return bson.M{"$and": cond}
}

func arrayElemFilter(array string, filterOfArray bson.M) bson.M {
	return bson.M{"$filter": bson.M{
		"input": fmt.Sprintf("$%s", array),
		"cond":  condOfArrayElemFilter(filterOfArray),
	}}
}

func errorIfMatchingNoDoc(r *mongo.UpdateResult) error {
	if r.MatchedCount == 0 {
		return dbmodels.DBError{
			ErrCode: util.ErrNoDBRecord,
			Err:     fmt.Errorf("doesn't match any records"),
		}
	}
	return nil
}

func (this *client) pushArrayElem(ctx context.Context, collection, array string, filterOfDoc, value bson.M) error {
	update := bson.M{"$push": bson.M{array: value}}

	col := this.collection(collection)
	r, err := col.UpdateOne(ctx, filterOfDoc, update)
	if err != nil {
		return err
	}

	return errorIfMatchingNoDoc(r)
}

func (this *client) pushArrayElems(ctx context.Context, collection, array string, filterOfDoc bson.M, value bson.A) error {
	update := bson.M{"$push": bson.M{array: bson.M{"$each": value}}}

	col := this.collection(collection)
	r, err := col.UpdateOne(ctx, filterOfDoc, update)
	if err != nil {
		return err
	}

	return errorIfMatchingNoDoc(r)
}

func (this *client) pullArrayElem(ctx context.Context, collection, array string, filterOfDoc, filterOfArray bson.M) error {
	update := bson.M{"$pull": bson.M{array: filterOfArray}}

	col := this.collection(collection)
	r, err := col.UpdateOne(ctx, filterOfDoc, update)
	if err != nil {
		return err
	}

	return errorIfMatchingNoDoc(r)
}

func (this *client) pullAndReturnArrayElem(ctx context.Context, collection, array string, filterOfDoc, filterOfArray bson.M, result interface{}) error {
	col := this.collection(collection)
	r := col.FindOneAndUpdate(
		ctx, filterOfDoc,
		bson.M{"$pull": bson.M{array: filterOfArray}},
		&options.FindOneAndUpdateOptions{
			Projection: bson.M{array: bson.M{"$elemMatch": filterOfArray}},
		})

	return r.Decode(result)
}

// r, _ := col.UpdateOne; r.ModifiedCount == 0 will happen in two case: 1. no matched array item; 2 update repeatedly with same update cmd.
// checkModified = true when it can't exclude any case of above two; otherwise it can be set as false.
func (this *client) updateArrayElem(ctx context.Context, collection, array string, filterOfDoc, filterOfArray, updateCmd bson.M, checkModified bool) error {
	cmd := bson.M{}
	for k, v := range updateCmd {
		cmd[fmt.Sprintf("%s.$[i].%s", array, k)] = v
	}

	arrayFilter := bson.M{}
	for k, v := range filterOfArray {
		arrayFilter["i."+k] = v
	}

	col := this.collection(collection)
	r, err := col.UpdateOne(
		ctx, filterOfDoc,
		bson.M{"$set": cmd},
		&options.UpdateOptions{
			ArrayFilters: &options.ArrayFilters{
				Filters: bson.A{
					arrayFilter,
				},
			},
		},
	)
	if err != nil {
		return err
	}

	if err := errorIfMatchingNoDoc(r); err != nil {
		return err
	}

	if r.ModifiedCount == 0 && checkModified {
		b, err := this.isArrayElemNotExists(ctx, collection, array, filterOfDoc, filterOfArray)
		if err == nil && b {
			return dbmodels.DBError{
				ErrCode: util.ErrNoDBRecord,
				Err:     fmt.Errorf("can't find array element"),
			}
		}
	}
	return nil
}

func (this *client) isArrayElemNotExists(ctx context.Context, collection, array string, filterOfDoc, filterOfArray bson.M) (bool, error) {
	query := bson.M{array: bson.M{"$elemMatch": filterOfArray}}
	for k, v := range filterOfDoc {
		query[k] = v
	}

	var v []struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	err := this.getDocs(ctx, collection, query, bson.M{"_id": 1}, &v)
	if err != nil {
		return false, err
	}

	return len(v) <= 0, nil
}

func (this *client) getArrayElem(ctx context.Context, collection, array string, filterOfDoc, filterOfArray, project bson.M, result interface{}) error {
	ma := map[string]bson.M{}
	if len(filterOfArray) > 0 {
		ma[array] = filterOfArray
	}
	return this.getMultiArrays(ctx, collection, filterOfDoc, ma, project, result)
}

func (this *client) getMultiArrays(ctx context.Context, collection string, filterOfDoc bson.M, filterOfArrays map[string]bson.M, project bson.M, result interface{}) error {
	pipeline := bson.A{bson.M{"$match": filterOfDoc}}

	if len(filterOfArrays) > 0 {
		project1 := bson.M{}

		for array, filterOfArray := range filterOfArrays {
			project1[array] = bson.M{"$filter": bson.M{
				"input": fmt.Sprintf("$%s", array),
				"cond":  condOfArrayElemFilter(filterOfArray),
			}}
		}

		for k, v := range project {
			s := k
			if i := strings.Index(k, "."); i >= 0 {
				s = k[:i]
			}
			if _, ok := filterOfArrays[s]; !ok {
				project1[k] = v
			}
		}

		pipeline = append(pipeline, bson.M{"$project": project1})
	}

	if len(project) > 0 {
		pipeline = append(pipeline, bson.M{"$project": project})
	}

	col := this.collection(collection)
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}

	return cursor.All(ctx, result)
}

func (this *client) countArray(ctx context.Context, collection, array string, filterOfDoc, filterOfArray bson.M) (int, error) {
	pipeline := bson.A{
		bson.M{"$match": filterOfDoc},
		bson.M{"$project": bson.M{
			"num": bson.M{"$cond": bson.A{
				bson.M{"$isArray": "$" + array},
				bson.M{"$size": arrayElemFilter(array, filterOfArray)},
				0,
			}},
		}},
	}

	col := this.collection(collection)
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}

	var result []struct {
		Num int `bson:"num"`
	}
	if err = cursor.All(ctx, result); err != nil {
		return 0, err
	}

	num := 0
	for _, item := range result {
		num += item.Num
	}
	return num, nil
}

func (this *client) newDocIfNotExist(ctx context.Context, collection string, filterOfDoc, docInfo bson.M) (string, error) {
	upsert := true

	col := this.collection(collection)
	r, err := col.UpdateOne(
		ctx, filterOfDoc, bson.M{"$setOnInsert": docInfo},
		&options.UpdateOptions{Upsert: &upsert},
	)
	if err != nil {
		return "", err
	}

	if r.UpsertedID == nil {
		return "", dbmodels.DBError{
			ErrCode: util.ErrRecordExists,
			Err:     fmt.Errorf("the doc exists"),
		}
	}

	return toUID(r.UpsertedID)
}

func (this *client) newDoc(ctx context.Context, collection string, filterOfDoc, docInfo bson.M) (string, error) {
	upsert := true

	col := this.collection(collection)
	r, err := col.ReplaceOne(
		ctx, filterOfDoc, docInfo,
		&options.ReplaceOptions{Upsert: &upsert},
	)
	if err != nil {
		return "", err
	}

	if r.UpsertedID != nil {
		return toUID(r.UpsertedID)
	}
	return "", nil
}

func (this *client) updateDoc(ctx context.Context, collection string, filterOfDoc, update bson.M) error {
	col := this.collection(collection)
	r, err := col.UpdateOne(ctx, filterOfDoc, bson.M{"$set": update})
	if err != nil {
		return err
	}
	return errorIfMatchingNoDoc(r)
}

func (this *client) getDoc(ctx context.Context, collection string, filterOfDoc, project bson.M, result interface{}) error {
	col := this.collection(collection)

	var sr *mongo.SingleResult
	if len(project) > 0 {
		sr = col.FindOne(ctx, filterOfDoc, &options.FindOneOptions{
			Projection: project,
		})
	} else {
		sr = col.FindOne(ctx, filterOfDoc)
	}

	if err := sr.Decode(result); err != nil {
		if err == mongo.ErrNoDocuments {
			return dbmodels.DBError{
				ErrCode: util.ErrNoDBRecord,
				Err:     fmt.Errorf("can't find record"),
			}
		}
		return err
	}
	return nil
}

func (this *client) getDocs(ctx context.Context, collection string, filterOfDoc, project bson.M, result interface{}) error {
	col := this.collection(collection)

	var cursor *mongo.Cursor
	var err error
	if len(project) > 0 {
		cursor, err = col.Find(ctx, filterOfDoc, &options.FindOptions{
			Projection: project,
		})
	} else {
		cursor, err = col.Find(ctx, filterOfDoc)
	}

	if err != nil {
		return err
	}
	return cursor.All(ctx, result)
}

func (this *client) insertDoc(ctx context.Context, collection string, docInfo bson.M) (string, error) {
	col := this.collection(collection)
	r, err := col.InsertOne(ctx, docInfo)
	if err != nil {
		return "", err
	}

	return toUID(r.InsertedID)
}
