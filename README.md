# Nyo

## Manage deployment from terminal easily

Access the TUI

```shell
nyo
```

Connect to your master node
--as is used to give an alias in case you have multiple master node using nyo

```shell
nyo connect master@ip --as something
```

Use any of the connections you made
all the other nyo commands will use that master node

```shell
nyo use something
```

Deploy the current project based on the instructions of the root Nyo.toml file

```shell
nyo deploy
```
