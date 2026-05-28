# 概述
此文档介绍 节点部署工作完成后，版本更新的手动操作
## 通过后台页面完成初始化数据库
访问8082页面，配置数据库，数据库IP使用私网IP
## 迁移后台用户
``` txt
上传 scripts/export_owmer_user.sh 脚本到目标数据库节点
下载导出的owner-users-export_*文件夹
将scripts/import_owner_user.sh 脚本拷贝放入owner-users-export_*文件夹 
将文件夹上传目标数据库节点
如果数据库名称更改运行前修改import_owner_user.sh 脚本中的数据库名称
在目标数据库节点执行 import_owner_user.sh 脚本
```
