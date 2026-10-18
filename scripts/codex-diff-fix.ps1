# Fix Codex diff on Windows (long git refs + Cursor webview render).
$repoRoot = Split-Path $PSScriptRoot -Parent
$extRoot = Join-Path $env:USERPROFILE ".cursor\extensions\openai.chatgpt-*-win32-x64"
$extDir = (Get-Item $extRoot -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending | Select-Object -First 1).FullName

git -C $repoRoot config core.longpaths true 2>$null
git config --global core.longpaths true 2>$null
Write-Host "Git core.longpaths enabled."

if (-not $extDir) {
    Write-Warning "Codex extension folder not found under $extRoot"
    exit 1
}

$editorDiff = Join-Path $extDir "webview\assets\editor-diff-page-AYqiUD9X.js"
$appInitial = Join-Path $extDir "webview\assets\app-initial-Bc0jVuU9.js"

function Patch-File($path, $old, $new, $label) {
    if (-not (Test-Path $path)) { Write-Warning "Missing $path"; return }
    $text = [IO.File]::ReadAllText($path)
    if ($text.Contains($new)) { Write-Host "OK (already patched): $label"; return }
    if (-not $text.Contains($old)) { Write-Warning "Pattern not found: $label"; return }
    [IO.File]::WriteAllText($path, $text.Replace($old, $new))
    Write-Host "Patched: $label"
}

Patch-File $editorDiff 'if(i?.unifiedDiff&&i.conversationId)try{let t=i.conversationId??null,n;' 'if(i?.unifiedDiff)try{let t=i.conversationId??null,n;' 'editor-diff: allow diff without conversationId'
Patch-File $editorDiff 'u;t[2]===a?u=t[3]:(u=r(a),t[2]=a,t[3]=u);let d=u' 'u;t[2]===a?u=t[3]:(u=(()=>{try{return r(a)}catch(e){return[]}})(),t[2]=a,t[3]=u);let d=u' 'editor-diff: safe parse'
Patch-File $editorDiff 'm;t[4]===d?m=t[5]:(m=ce(d),t[4]=d,t[5]=m);' 'm;t[4]===d?m=t[5]:(m=(()=>{try{return ce(d)}catch(e){return{fileCount:0,linesAdded:0,linesDeleted:0}}})(),t[4]=d,t[5]=m);' 'editor-diff: safe summary'
# Repair older broken IIFE patches (invalid `}}()` invoke)
Patch-File $editorDiff '(()=>{try{u=r(a)}catch(e){u=[]}}(),t[2]=a,t[3]=u)' 'u=(()=>{try{return r(a)}catch(e){return[]}})(),t[2]=a,t[3]=u' 'editor-diff: repair broken parse IIFE'
Patch-File $editorDiff '(()=>{try{u=r(a)}catch(e){u=[]}})(),t[2]=a,t[3]=u)' 'u=(()=>{try{return r(a)}catch(e){return[]}})(),t[2]=a,t[3]=u' 'editor-diff: repair half-fixed parse IIFE'
Patch-File $editorDiff '(()=>{try{m=ce(d)}catch(e){m={fileCount:0,linesAdded:0,linesDeleted:0}}}(),t[4]=d,t[5]=m)' 'm=(()=>{try{return ce(d)}catch(e){return{fileCount:0,linesAdded:0,linesDeleted:0}}})(),t[4]=d,t[5]=m' 'editor-diff: repair broken summary IIFE'
Patch-File $editorDiff '(()=>{try{m=ce(d)}catch(e){m={fileCount:0,linesAdded:0,linesDeleted:0}}})(),t[4]=d,t[5]=m)' 'm=(()=>{try{return ce(d)}catch(e){return{fileCount:0,linesAdded:0,linesDeleted:0}}})(),t[4]=d,t[5]=m' 'editor-diff: repair half-fixed summary IIFE'
# Keep disk fallback enabled (disabling it often yields empty/broken diffs on refactors)
Patch-File $editorDiff 'fullContentNextFallbackToDisk:!1' 'fullContentNextFallbackToDisk:!0' 'editor-diff: restore disk fallback'
Patch-File $editorDiff 'hostConfig:y,conversationId:o' 'hostConfig:y??{},conversationId:o' 'editor-diff: hostConfig fallback'
Patch-File $editorDiff 'initialOpen:k&&e.metadata.type!==`deleted`' 'initialOpen:!1' 'editor-diff: no auto-open deleted'
Patch-File $appInitial 'nextFallbackToDisk:!1' 'nextFallbackToDisk:p===`branch`||p===`unstaged`' 'thread diff: restore disk fallback'

Write-Host ""
Write-Host "Done. Reload Cursor: Developer -> Reload Window, then open Codex diff again."
Write-Host "Extension path: $extDir"
