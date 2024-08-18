# Database Transaction

Transactions are a fundamental concept of all database systems. The essential point of a transaction is that it bundles multiple steps into a single, all-or-nothing operation. The intermediate states between the steps are not visible to other concurrent transactions, and if some failure occurs that prevents the transaction from completing, then none of the steps affect the database at all.

## Mango SQL Transaction

To make developer life easier, Mango SQL natively support transaction.

::: code-group

```go [With Transaction]
// if one request fail, both will be rollback
err := db.Transaction(func(tx *DBClient) error {
    _, err1 := tx.User.Upsert(UserUpdate{Id: 1, Name: "usernew"})
	_, err2 = tx.User.Upsert(UserUpdate{Id: 2, Name: "user1-updated"})
    return errors.Join(err1, err2)
})
```

```go [Without Transaction]
// if one request fail, the other may still modify the database
_, err1 := db.User.Upsert(UserUpdate{Id: 1, Name: "usernew"})
_, err2 = db.User.Upsert(UserUpdate{Id: 2, Name: "user1-updated"})
return errors.Join(err1, err2)
```

:::

## Reusable Parameter

To make code more reusable, Mango SQL use the same object `*DBClient` for normal and transaction.
No need to handle multiple signature like `sql.Db` and `sql.Tx`

```go 
func AppendLog(db *DBClient, msg string) error {
    _, err := db.Log.Insert(LogCreate{ Msg: msg })
    return err
}

func main() {
    // can be called directly with db
    AppendLog(db, "Hello")

    err := db.Transaction(func(tx *DBClient) error {
        // also accept to be called in a transaction
        return AppendLog(tx, "Hello")
    })
}
```

::: warning

Nested Transactions are not supported

:::