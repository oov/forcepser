-- ファイルに変更があったときに呼ばれる関数
function changed(files, trycount, proj)
  local success = {}
  for i, file in ipairs(files) do
    if trycount[i] == 0 then
      debug_print(file)
    else
      debug_print(file .. " " .. (trycount[i]+1) .. "回目")
    end
    local rule, text, outfile = findrule(file, proj)
    if rule ~= nil then
      debug_print_verbose("ルールに一致: " .. rule.file .. " / 挿入先レイヤー: " .. rule.layer)
      local ok, err = pcall(drop, proj, outfile, text, rule)
      if ok then
        table.insert(success, file)
        debug_print("  レイヤー " .. rule.layer .. " へドロップしました")
      else
        debug_print("  処理中にエラーが発生しました: " .. err)
      end
    else
      debug_print("  一致するルールが見つかりませんでした")
      table.insert(success, file)
    end
  end
  return success
end

local function genexo(proj, file, text, rule)
  local ai = getaudioinfo(file)
  local length = math.floor((ai.samples * proj.video_rate) / (ai.samplerate * proj.video_scale))
  local padding = math.floor((rule.padding * proj.video_rate) / (1000 * proj.video_scale))
  local exo = {}
  table.insert(exo, "[exedit]")
  table.insert(exo, "width=" .. proj.width)
  table.insert(exo, "height=" .. proj.height)
  table.insert(exo, "rate=" .. proj.video_rate)
  table.insert(exo, "scale=" .. proj.video_scale)
  table.insert(exo, "length=" .. length)
  table.insert(exo, "audio_rate=" .. proj.audio_rate)
  table.insert(exo, "audio_ch=" .. proj.audio_ch)
  table.insert(exo, "[0]")
  table.insert(exo, "start=1")
  table.insert(exo, "end=" .. length)
  table.insert(exo, "layer=1")
  table.insert(exo, "group=1")
  table.insert(exo, "overlay=1")
  table.insert(exo, "audio=1")
  table.insert(exo, "[0.0]")
  table.insert(exo, "_name=音声ファイル")
  table.insert(exo, "再生位置=0.00")
  table.insert(exo, "再生速度=100.0")
  table.insert(exo, "ループ再生=0")
  table.insert(exo, "動画ファイルと連携=0")
  table.insert(exo, "file=" .. file)
  table.insert(exo, "__json=" .. toexostring('{"padding":'..padding..'}'))
  table.insert(exo, "[0.1]")
  table.insert(exo, "_name=標準再生")
  table.insert(exo, "音量=100.0")
  table.insert(exo, "左右=0.0")
  table.insert(exo, "[1]")
  table.insert(exo, "start=1")
  table.insert(exo, "end=" .. length)
  table.insert(exo, "layer=2")
  table.insert(exo, "group=1")
  table.insert(exo, "overlay=1")
  table.insert(exo, "camera=0")
  table.insert(exo, "[1.0]")
  table.insert(exo, "_name=テキスト")
  table.insert(exo, "サイズ=24")
  table.insert(exo, "表示速度=0.0")
  table.insert(exo, "文字毎に個別オブジェクト=0")
  table.insert(exo, "移動座標上に表示する=0")
  table.insert(exo, "自動スクロール=0")
  table.insert(exo, "B=0")
  table.insert(exo, "I=0")
  table.insert(exo, "type=0")
  table.insert(exo, "autoadjust=0")
  table.insert(exo, "soft=0")
  table.insert(exo, "monospace=0")
  table.insert(exo, "align=4")
  table.insert(exo, "spacing_x=0")
  table.insert(exo, "spacing_y=0")
  table.insert(exo, "precision=0")
  table.insert(exo, "color=ffffff")
  table.insert(exo, "color2=000000")
  table.insert(exo, "font=MS UI Gothic")
  table.insert(exo, "text=" .. toexostring(text))
  table.insert(exo, "[1.1]")
  table.insert(exo, "_name=標準描画")
  table.insert(exo, "X=0.0")
  table.insert(exo, "Y=0.0")
  table.insert(exo, "Z=0.0")
  table.insert(exo, "拡大率=100.00")
  table.insert(exo, "透明度=0.0")
  table.insert(exo, "回転=0.00")
  table.insert(exo, "blend=0")
  return tosjis(table.concat(exo, "\r\n")), length+padding
end

local function genexofromtemplate(exo, proj, file, text, rule)
  local f, err = io.open(exo, "rb")
  if f == nil then
    return nil
  end
  local s = fromsjis(f:read("*all"))
  f:close()
  local ai = getaudioinfo(file)
  local length = math.floor((ai.samples * proj.video_rate) / (ai.samplerate * proj.video_scale))
  local padding = math.floor((rule.padding * proj.video_rate) / (1000 * proj.video_scale))
  s = s:gsub("%%WIDTH%%", tostring(proj.width))
  s = s:gsub("%%HEIGHT%%", tostring(proj.height))
  s = s:gsub("%%RATE%%", tostring(proj.video_rate))
  s = s:gsub("%%SCALE%%", tostring(proj.video_scale))
  s = s:gsub("%%LENGTH%%", tostring(length))
  s = s:gsub("%%PADDING%%", tostring(padding))
  s = s:gsub("%%AUDIO_RATE%%", tostring(proj.audio_rate))
  s = s:gsub("%%AUDIO_CH%%", tostring(proj.audio_ch))
  s = s:gsub("%%WAVE%%", file)
  s = s:gsub("%%TEXT%%", text)
  s = s:gsub("%%EXOTEXT%%", toexostring(text))
  s = s:gsub("%%EXOJSON%%", toexostring('{"padding":'..padding..'}'))
  return tosjis(s), length+padding
end

function drop(proj, file, text, rule)
  local exo, length = nil, nil
  local f, err = loadfile(rule.luafile)
  if f ~= nil then
    local m = f()
    if m.gen2 ~= nil then
      exo, length = m.gen2(proj, file, text, rule)
    elseif m.gen ~= nil then
      _G['exofile'] = rule.exofile
      _G['luafile'] = rule.luafile
      exo, length = m.gen(proj, file, text, rule.layer, rule.userdata)
    end
  end
  if exo == nil then
    exo, length = genexofromtemplate(rule.exofile, proj, file, text, rule)
  end
  if exo == nil then
    exo, length = genexo(proj, file, text, rule)
  end
  f, err = io.open("temp.exo", "wb")
  if f == nil then
    error("exo ファイルが作成できません: " .. err)
  end
  f:write(exo)
  f:close()
  sendfile(proj.window, rule.layer, length, {"temp.exo"})
end