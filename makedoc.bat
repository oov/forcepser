git rm -rf docs/*
mkdir docs
cd ../forcepser_config_html
call pack.bat
cp -r build/* ../forcepser/docs/
cd ../forcepser
git add docs
