# OpenMPRDB-CLI

API文档: https://openmprdb.org/

计划: https://www.processon.com/view/link/61115e3ae401fd5eeb80a3a9

## 编译

**该项目需要本地安装Golang(>=1.16)和GCC环境**

```shell
git clone https://github.com/wsndshx/OpenMPRDB-CLI.git
cd OpenMPRDB-CLI
go mod download
go build
```

## 使用说明

在首次运行时, 会在程序所在目录下生成密钥, 公钥, 数据库文件, 因此请将程序放置在单独的文件夹中运行, 避免污染公共目录.

### 注册

```shell
OpenMPRDB-CLI -register --server_name Neko
```
- `server_name` 服务器名称

### 提交新声望数据

```shell
OpenMPRDB-CLI --player 252af321-89aa-426c-a534-399f551810ae --point "1" --comment "因为喜欢"
```

- `player` 玩家的 UUID

- `point` 评分, 取值在 -1 ~ 1 之间

- `comment` 关于打分的说明

### 撤回(删除)之前的提交

```shell
delete -submit cc472483-ed6b-4204-a19c-d268decc7730 -comment "不喜欢了"
```

- `submit` 需要撤回的提交的操作 uuid (可查询)

- `comment` 撤回的理由

### 导入其他服务器的公钥并信任

```shell
import -uuid cc91e632-5636-4cac-b0b0-6508a35aede4 -pubkey "D:\OpenMPRDB-CLI\rsa-pub.pem"
```

- `uuid` 需要导入的服务器在中心服务器中的 uuid

- `pubkey` 需要导入的服务器的公钥路径

### 根据本地提交数据和导入的其他服务器提交数据生成玩家声誉报告(默认列出所有玩家)

```shell
OpenMPRDB-CLI update -less "-0.1" -export "./banned-players.json"
```

- `less` 只输出小于等于该值的玩家

- `export` 将结果导出到指定文件(输出格式为[该页](https://minecraft.fandom.com/de/wiki/Befehl/ban)所定义的格式)

### 列表

为了偷懒和方便, 程序把服务器信息和提交信息存放在了数据库文件中, 并且提供了一个简易的列表功能.

#### 列出当前信任的服务器(包含自身)数据(不包含公钥)

```shell
OpenMPRDB-CLI list server
```

此操作会列出当前信任的服务器的 uuid 和名称.

#### 列出提交记录

```shell
/OpenMPRDB-CLI list sub
```

此操作会列出过去提交的详细信息, 包含操作uuid.