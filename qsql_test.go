package omgsql

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGetPositionsListByOneStruct(t *testing.T) {

	data := struct {
		F1 int `db:"f1"`
		F3 int `db:"f2"`
		F2 int `db:"f3"`
	}{}

	columns := []string{"f1", "f2", "f3"}

	dest := []interface{}{&data}

	pos, err := getPositionsList(columns, dest)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pos, [][][]int{{{0}, {1}, {2}}}) {
		t.Fatal(err)
	}
}

func TestGetPositionsListBySomeStruct(t *testing.T) {

	data1 := struct {
		F1 int `db:"f1"`
		F3 int `db:"f2"`
		F2 int `db:"f3"`
	}{}

	data2 := struct {
		F4 int `db:"f4"`
		F5 int `db:"f5"`
	}{}

	columns := []string{"f1", "f3", "f2", "f5", "f4"}

	dest := []interface{}{&data1, &data2}

	pos, err := getPositionsList(columns, dest)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pos, [][][]int{{{0}, {2}, {1}}, {{1}, {0}}}) {
		t.Fatal(err)
	}
}

func TestGetPositionsListByComplexStruct1(t *testing.T) {

	type Data1 struct {
		F1 int `db:"f1"`
		F3 int `db:"f2"`
		F2 int `db:"f3"`
	}

	type Data2 struct {
		F4 int `db:"f4"`
		F5 int `db:"f5"`
	}

	data := struct {
		Data1
		Data2
	}{}

	columns := []string{"f1", "f3", "f2", "f5", "f4"}

	dest := []interface{}{&data}

	pos, err := getPositionsList(columns, dest)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pos, [][][]int{{{0, 0}, {0, 2}, {0, 1}, {1, 1}, {1, 0}}}) {
		t.Fatal(err)
	}
}

func TestGetPositionsListByComplexStruct2(t *testing.T) {

	type Data1 struct {
		F1 int `db:"f1"`
		F3 int `db:"f2"`
		F2 int `db:"f3"`
	}

	data := struct {
		Data1

		F4 int `db:"f4"`
		F5 int `db:"f5"`
	}{}

	columns := []string{"f1", "f3", "f2", "f5", "f4"}

	dest := []interface{}{&data}

	pos, err := getPositionsList(columns, dest)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pos)
	if !reflect.DeepEqual(pos, [][][]int{{{0, 0}, {0, 2}, {0, 1}, {2}, {1}}}) {
		t.Fatal(err)
	}
}

func TestGetPositionsListByComplexStruct3(t *testing.T) {

	data := struct {
		Q struct {
			F1 int `db:"f1"`
			F3 int `db:"f2"`
			F2 int `db:"f3"`
		} `db:"*"`

		F4 int `db:"f4"`
		F5 int `db:"f5"`
	}{}

	columns := []string{"f1", "f3", "f2", "f5", "f4"}

	dest := []interface{}{&data}

	pos, err := getPositionsList(columns, dest)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pos, [][][]int{{{0, 0}, {0, 2}, {0, 1}, {2}, {1}}}) {
		t.Fatal(err)
	}
}
