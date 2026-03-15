# create.ps1
$name = Read-Host "migrate name"

if ($name) {
    migrate create -ext sql -dir migrations $name
    Write-Host "[changed] migrate created"
}
else {
    Write-Host "[no changed] migrate name is empty"
}