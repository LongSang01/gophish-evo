# Gophish-Evo

基于 [Gophish](https://github.com/gophish/gophish) 的二次开发版本，进行中文本地化和功能增强。

在原版 Gophish 基础上修改以下功能：

### 主要更新

- **中文本地化**：改用基于[Vben Admin](https://github.com/vbenjs/vue-vben-admin)修改的全新前端页面
- **多发件箱支持**：支持调用多个发件箱在单个任务内均等发送
- **二维码功能**：内置二维码生成，补丁来自[gophish-z](https://github.com/hikeny666/gophish-z)
- **JWT Token**：修改原本的`Cookie`认证为`JWT Token`认证

### 修改项

- 将"姓"和"名"合并为"姓名"单列，符合中文习惯
- 修复复制发件箱时 header 头不被复制的问题
- 修改默认监听地址改为 127.0.0.1:8080，推荐使用 nginx 反向代理
- 更新部分依赖项

### 构建

- 后端代码

```bash
git clone https://github.com/LongSang01/gophish-evo
cd gophish-evo
go build
```

- 前端代码

```bash
cd web
npm install
npm run build
```

### 文档

文档请参考 [Gophish 官方文档](https://getgophish.com/documentation/)

### Thanks

https://github.com/gophish/gophish

https://github.com/hikeny666/gophish-z

https://github.com/vbenjs/vue-vben-admin

### 免责声明

本项目仅用于授权的安全意识培训、红队演练及渗透测试场景。
使用者需确保在获得明确授权的前提下使用本工具,
对任何未经授权的使用所造成的后果,作者不承担任何责任
