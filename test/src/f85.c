long f(long n) {
  long sum = 0;
  long i;
  
  for (i = 0; i < n; i = i + 1) {
    long temp = i * i;
    sum = sum + temp;
  }
  
  return sum;
} 