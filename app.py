print("Basit Hesap Makinesi")

# Kullanıcıdan veri alma
sayi1 = float(input("Birinci sayıyı girin: "))
sayi2 = float(input("İkinci sayıyı girin: "))

print("\nİşlemler:")
print("1 - Toplama (+)")
print("2 - Çıkarma (-)")
print("3 - Çarpma (*)")
print("4 - Bölme (/)")

secim = input("Yapmak istediğiniz işlemi seçin (1/2/3/4): ")

if secim == "1":
    print("Sonuç:", sayi1 + sayi2)
elif secim == "2":
    print("Sonuç:", sayi1 - sayi2)
elif secim == "3":
    print("Sonuç:", sayi1 * sayi2)
elif secim == "4":
    if sayi2 != 0:
        print("Sonuç:", sayi1 / sayi2)
    else:
        print("Hata: Bir sayı 0'a bölünemez!")
else:
    print("Geçersiz seçim yaptınız!")
