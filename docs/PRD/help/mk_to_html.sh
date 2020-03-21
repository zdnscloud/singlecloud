#!/bin/bash

rm -rf preview
i5ting_toc -f usermanual.md 
cp -rf *.jpg *.png preview/
sed -i 's/i5ting_ztree_toc:usermanual/Zcloud操作手册/g' preview/usermanual.html
mv preview/usermanual.html preview/index.html
sed -i 's/Table of Content/Zcloud操作手册/g' preview/toc/js/*.js
rm -rf help
mv preview help
