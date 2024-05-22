# Startup the WinRM service
Enable-PSRemoting -Force 

# Create a Firewall rule to allow build computer to connect to the Azure VM
New-NetFirewallRule -Name "Allow WinRM HTTPS" -DisplayName "WinRM HTTPS" -Enabled True -Profile Any -Action Allow -Direction Inbound -LocalPort 5986 -Protocol TCP

# Used for creating the WinRM certificate for authentication
$thumbprint = (New-SelfSignedCertificate -DnsName $env:COMPUTERNAME -CertStoreLocation Cert:\LocalMachine\My -NotAfter $(Get-Date).AddDays(1)).Thumbprint 

# Create a new WinRM listener using this certificate
$command = "winrm create winrm/config/Listener?Address=*+Transport=HTTPS @{Hostname=""$env:computername""; CertificateThumbprint=""$thumbprint""}" 
cmd.exe /C $command
