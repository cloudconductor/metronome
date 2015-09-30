概要
=====

MetronomeはConsulクラスタメンバ間のイベント処理について順序制御を行うツールです。

Consulではイベントがクラスタに送信されると、クラスタ内のサーバが一斉に処理を開始します。しかし、複数のサーバで構成されたシステムにおいては、処理の順番が重要になることがあります。たとえば、データベースサーバはアプリケーションサーバよりも先に初期化を終えていなくてはなりませんし、ロードバランサーは全てのサーバが準備できてからクライアントのリクエストを受け付けるべきでしょう。

Metronomeは設定ファイルとConsul KVSを用いることで、これらの順序制御を行うことができます。

![Metronomeのアーキテクチャ](https://raw.githubusercontent.com/wiki/cloudconductor/metronome/ja/diagram.png)

Consulイベントを受信すると、Consulのイベントハンドラによってmetronome pushが起動され、Consul KVS上のイベントキューに受信したイベントが追加されます。
Metronomeはイベントキューからイベントをポーリングし、設定ファイルで指定されたサービス名、タグ名を条件として追加した上で実行タスクキューにイベントを格納します。
各サーバは実行タスクキューの先頭タスクを取得し、条件を満たした場合には処理を行います。他のサーバは先頭タスクを取得しながらこれらの処理が終了するのを待機します。

- [User Manual(ja)](https://github.com/cloudconductor/metronome/wiki/User-Manual(ja))
- [Scheduling file format(ja)](https://github.com/cloudconductor/metronome/wiki/Scheduling-file-format(ja))

前提条件
============

システム
-------------------

- OS: Red Hat Enterprise Linux 6.5 or CentOS 6.5

依存関係
-------------

- consul

Copyright and License
=====================

Copyright 2015 TIS inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


Contact
========

For more information: <http://cloudconductor.org/>

Report issues and requests: <https://github.com/cloudconductor/cloud_conductor/issues>

Send feedback to: <ccndctr@gmail.com>
