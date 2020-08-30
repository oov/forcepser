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
  local length = math.ceil((ai.samples * proj.video_rate) / (ai.samplerate * proj.video_scale))
  local padding = math.ceil((rule.padding * proj.video_rate) / (1000 * proj.video_scale))
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

local function parseexo(lines)
  local ini = {}
  local sect = ""
  for line in lines:gmatch('[^\r\n]+') do
    local m = line:match('^%[([^%]]+)%]$')
    if m ~= nil then
      sect = m
      ini[sect] = ini[sect] or {}
    else
      local k, v = line:match('^([^=]+)=(.*)$')
      if k ~= nil then
        ini[sect][k] = v
      end
    end
  end
  return ini
end

local function genexofromtemplate(exo, proj, file, text, rule)
  local f, err = io.open(exo, "rb")
  if f == nil then
    return nil
  end
  local s = fromsjis(f:read("*all"))
  f:close()
  debug_print("  テンプレートファイル " .. exo .. " を使用します")
  local ai = getaudioinfo(file)
  if s:match('%%WAVE%%') ~= nil then
    local length = math.ceil((ai.samples * proj.video_rate) / (ai.samplerate * proj.video_scale))
    local padding = math.ceil((rule.padding * proj.video_rate) / (1000 * proj.video_scale))
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
  else
    local ini = parseexo(s)
    local tplproj = {
      width = tonumber(ini.exedit.width),
      height = tonumber(ini.exedit.height),
      video_rate = tonumber(ini.exedit.rate),
      video_scale = tonumber(ini.exedit.scale),
      audio_rate = tonumber(ini.exedit.audio_rate),
      audio_ch = tonumber(ini.exedit.audio_ch),
    }
    local length = math.ceil((ai.samples * tplproj.video_rate) / (ai.samplerate * tplproj.video_scale))
    local padding = math.ceil((rule.padding * tplproj.video_rate) / (1000 * tplproj.video_scale))
    local tgst, tged = -1, -1
    for sect, t in pairs(ini) do
      if sect:match('^%d+%.0$') ~= nil then
        if t._name == "音声ファイル" and t.file == "" then
          -- ファイルを指定していない音声ファイルオブジェクトには音声ファイルへのパスを突っ込む
          ini[sect].file = file
          ini[sect]["動画ファイルと連携"] = "0"
          ini[sect].__json = toexostring('{"padding":'..padding..'}')
          local psect = sect:sub(1, #sect-2)
          tgst, tged = ini[psect]["start"], ini[psect]["end"]
        elseif t._name == "テキスト" and t.text:sub(1, 12) == "575b555e0000" then
          -- 本文が「字幕」になっているテキストオブジェクトには字幕を突っ込む
          ini[sect].text = toexostring(text)
        end
      end
    end
    for sect, t in pairs(ini) do
      if sect:match('^%d+$') ~= nil then
        if t["start"] == tgst and t["end"] == tged then
          ini[sect]["end"] = tostring(length)
        elseif t["end"] == tged then
          ini[sect]["start"] = tostring(t["start"] + length - tged)
          ini[sect]["end"] = tostring(length)
        end
      end
    end
    local r = {}
    for sect, t in pairs(ini) do
      table.insert(r, "[" .. sect .. "]")
      for k, v in pairs(t) do
        table.insert(r, k .. "=" .. v)
      end
    end
    return tosjis(table.concat(r, "\r\n")), length+padding
  end
end

function drop(proj, file, text, rule)
  rule.luafile = replaceenv(rule.luafile)
  rule.exofile = replaceenv(rule.exofile)
  local exo, length = nil, nil
  local f, err = loadfile(rule.luafile)
  if f ~= nil then
    local m = f()
    if m.gen2 ~= nil or m.gen ~= nil then
      debug_print("  テンプレートスクリプト " .. rule.luafile .. " を使用します")
    end
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