# Zappac

Zappac is a calculator, for people who think like programmers.

```
1 + 2
$foo = 1 + 2
$bar = ($foo - 1) ** 16
1024 ^ $foo + $bar
```

More documentation probably coming at some point.

![Demo](./zappac.gif)

Press `ctrl+f` to toggle number formatting.

## Running

```shell
go run zappac.go
# or
go build -o zappac
./zappac
```

If you want to set up your development version in your system, run

```shell
go install
```

Ensure `$GOPATH/bin` is in your `$PATH`.
