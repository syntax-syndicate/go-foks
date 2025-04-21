- apt get install pcscd
-   cat /etc/polkit-1/rules.d/90-pcscd.rules 
polkit.addRule(function(action, subject) {
    if (action.id == "org.debian.pcsc-lite.access_pcsc" &&
        subject.isInGroup("pcscd")) {
        return polkit.Result.YES;
    }
});

- groupadd pcscd
- usermod -aG pcscd max 
- systemctl daemon-reload
- systemctl restart polkit
- systemctl restart pcscd

