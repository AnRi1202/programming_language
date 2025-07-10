long f(long x) {
  long arr[3];
  long* ptr;
  long i;
  long sum;
  
  arr[0] = x;
  arr[1] = x * 2;
  arr[2] = x * 3;
  
  ptr = arr;
  sum = 0;
  
  for (i = 0; i < 3; i = i + 1) {
    sum = sum + ptr[i];
  }
  
  return sum;
} 