 #!/bin/bash
 # Script with conditional logic
 VALUE={{.value}}
 if [ $VALUE -gt 10 ]; then
   echo "Value $VALUE is greater than 10"
 else
   echo "Value $VALUE is less than or equal to 10"
 fi
